package gasoutput

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	derivedmodel "github.com/filecoin-project/lily/model/derived"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/messages"
)

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{
		node: node,
	}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "gas_outputs"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	grp, grpCtx := errgroup.WithContext(ctx)

	var getActorCodeFn func(address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(grpCtx, t.node.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = t.node.TipSetMessageReceipts(grpCtx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})
	var burnFn lens.ShouldBurnFn
	grp.Go(func() error {
		var err error
		burnFn, err = t.node.ShouldBrunFn(grpCtx, executed)
		if err != nil {
			return fmt.Errorf("getting should burn function: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		report.ErrorsDetected = err
		return nil, report, nil
	}

	var (
		gasOutputsResults = make(derivedmodel.GasOutputsList, 0)
		errorsDetected    = make([]*messages.MessageError, 0)
		exeMsgSeen        = make(map[cid.Cid]bool)
	)

	for _, msgrec := range blkMsgRec {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := msgrec.Iterator()
		if err != nil {
			return nil, nil, err
		}

		blk := msgrec.Block
		for itr.HasNext() {
			m, _, r := itr.Next()
			if exeMsgSeen[m.Cid()] {
				continue
			}
			exeMsgSeen[m.Cid()] = true

			toActorCode, found := getActorCodeFn(m.VMMessage().To)
			if !found {
				toActorCode = cid.Undef
			}
			gasOutputs, err := datasource.ComputeGasOutputs(ctx, blk, m.VMMessage(), r, burnFn)
			if err != nil {
				return nil, nil, err
			}
			actorName := builtin.ActorNameByCode(toActorCode)
			gasOutput := &derivedmodel.GasOutputs{
				Height:             int64(blk.Height),
				StateRoot:          blk.ParentStateRoot.String(),
				ParentBaseFee:      blk.ParentBaseFee.String(),
				Cid:                m.Cid().String(),
				From:               m.VMMessage().From.String(),
				To:                 m.VMMessage().To.String(),
				Value:              m.VMMessage().Value.String(),
				GasFeeCap:          m.VMMessage().GasFeeCap.String(),
				GasPremium:         m.VMMessage().GasPremium.String(),
				GasLimit:           m.VMMessage().GasLimit,
				Nonce:              m.VMMessage().Nonce,
				Method:             uint64(m.VMMessage().Method),
				SizeBytes:          m.ChainLength(),
				ExitCode:           int64(r.ExitCode),
				GasUsed:            r.GasUsed,
				BaseFeeBurn:        gasOutputs.BaseFeeBurn.String(),
				OverEstimationBurn: gasOutputs.OverEstimationBurn.String(),
				MinerPenalty:       gasOutputs.MinerPenalty.String(),
				MinerTip:           gasOutputs.MinerTip.String(),
				Refund:             gasOutputs.Refund.String(),
				GasRefund:          gasOutputs.GasRefund,
				GasBurned:          gasOutputs.GasBurned,
				ActorName:          actorName,
				ActorFamily:        builtin.ActorFamily(actorName),
			}
			gasOutputsResults = append(gasOutputsResults, gasOutput)
		}
	}

	if len(errorsDetected) > 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		gasOutputsResults,
	}, report, nil
}
