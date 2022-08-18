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

	grp, ctx := errgroup.WithContext(ctx)

	var getActorCodeFn func(address address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCodeFn, err = util.MakeGetActorCodeFunc(ctx, t.node.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("getting actor code lookup function: %w", err)
		}
		return nil
	})

	var blkMsgRec []*lens.BlockMessageReceipts
	grp.Go(func() error {
		var err error
		blkMsgRec, err = t.node.TipSetMessageReceipts(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages and receipts: %w", err)
		}
		return nil
	})
	var burnFn lens.ShouldBurnFn
	grp.Go(func() error {
		var err error
		burnFn, err = t.node.ShouldBrunFn(ctx, current, executed)
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
			m, r := itr.Next()
			if exeMsgSeen[m.Cid()] {
				continue
			}
			exeMsgSeen[m.Cid()] = true

			var msgSize int
			if b, err := m.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid(),
					Error: fmt.Errorf("failed to serialize message: %w", err).Error(),
				})
			}

			toActorCode, found := getActorCodeFn(m.To)
			if !found {
				toActorCode = cid.Undef
			}
			gasOutputs, err := datasource.ComputeGasOutputs(ctx, blk, m, r, burnFn)
			if err != nil {
				return nil, nil, err
			}
			actorName := builtin.ActorNameByCode(toActorCode)
			gasOutput := &derivedmodel.GasOutputs{
				Height:             int64(blk.Height),
				Cid:                m.Cid().String(),
				From:               m.From.String(),
				To:                 m.To.String(),
				Value:              m.Value.String(),
				GasFeeCap:          m.GasFeeCap.String(),
				GasPremium:         m.GasPremium.String(),
				GasLimit:           m.GasLimit,
				Nonce:              m.Nonce,
				Method:             uint64(m.Method),
				StateRoot:          blk.ParentStateRoot.String(),
				ExitCode:           int64(r.ExitCode),
				GasUsed:            r.GasUsed,
				ParentBaseFee:      blk.ParentBaseFee.String(),
				SizeBytes:          msgSize,
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
