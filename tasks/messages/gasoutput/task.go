package gasoutput

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/chain/actors/builtin"
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

	tsMsgs, err := t.node.ExecutedAndBlockMessages(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting executed and block messages: %w", err)
		return nil, report, nil
	}
	emsgs := tsMsgs.Executed

	var (
		gasOutputsResults = make(derivedmodel.GasOutputsList, 0, len(emsgs))
		errorsDetected    = make([]*messages.MessageError, 0, len(emsgs))
		exeMsgSeen        = make(map[cid.Cid]bool, len(emsgs))
	)

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		if exeMsgSeen[m.Cid] {
			continue
		}
		exeMsgSeen[m.Cid] = true

		var msgSize int
		if b, err := m.Message.Serialize(); err == nil {
			msgSize = len(b)
		} else {
			errorsDetected = append(errorsDetected, &messages.MessageError{
				Cid:   m.Cid,
				Error: fmt.Errorf("failed to serialize message: %w", err).Error(),
			})
		}

		actorName := builtin.ActorNameByCode(m.ToActorCode)
		gasOutput := &derivedmodel.GasOutputs{
			Height:             int64(m.Height),
			Cid:                m.Cid.String(),
			From:               m.Message.From.String(),
			To:                 m.Message.To.String(),
			Value:              m.Message.Value.String(),
			GasFeeCap:          m.Message.GasFeeCap.String(),
			GasPremium:         m.Message.GasPremium.String(),
			GasLimit:           m.Message.GasLimit,
			Nonce:              m.Message.Nonce,
			Method:             uint64(m.Message.Method),
			StateRoot:          m.BlockHeader.ParentStateRoot.String(),
			ExitCode:           int64(m.Receipt.ExitCode),
			GasUsed:            m.Receipt.GasUsed,
			ParentBaseFee:      m.BlockHeader.ParentBaseFee.String(),
			SizeBytes:          msgSize,
			BaseFeeBurn:        m.GasOutputs.BaseFeeBurn.String(),
			OverEstimationBurn: m.GasOutputs.OverEstimationBurn.String(),
			MinerPenalty:       m.GasOutputs.MinerPenalty.String(),
			MinerTip:           m.GasOutputs.MinerTip.String(),
			Refund:             m.GasOutputs.Refund.String(),
			GasRefund:          m.GasOutputs.GasRefund,
			GasBurned:          m.GasOutputs.GasBurned,
			ActorName:          actorName,
			ActorFamily:        builtin.ActorFamily(actorName),
		}
		gasOutputsResults = append(gasOutputsResults, gasOutput)
	}

	if len(errorsDetected) > 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		gasOutputsResults,
	}, report, nil
}
