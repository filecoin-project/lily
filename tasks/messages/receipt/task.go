package receipt

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
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
			attribute.String("processor", "receipts"),
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
		receiptResults = make(messagemodel.Receipts, 0, len(emsgs))
		errorsDetected = make([]*messages.MessageError, 0, len(emsgs))
		exeMsgSeen     = make(map[cid.Cid]bool, len(emsgs))
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

		rcpt := &messagemodel.Receipt{
			Height:    int64(current.Height()),
			Message:   m.Cid.String(),
			StateRoot: current.ParentState().String(),
			Idx:       int(m.Index),
			ExitCode:  int64(m.Receipt.ExitCode),
			GasUsed:   m.Receipt.GasUsed,
		}
		receiptResults = append(receiptResults, rcpt)

	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		receiptResults,
	}, report, nil
}
