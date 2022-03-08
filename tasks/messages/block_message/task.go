package block_message

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

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
			attribute.String("processor", "block_messages"),
		)
	}
	defer span.End()
	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	tsMsgs, err := t.node.ExecutedAndBlockMessages(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = xerrors.Errorf("getting executed and block messages: %w", err)
		return nil, report, nil
	}
	blkMsgs := tsMsgs.Block

	var (
		errorsDetected      = make([]*messages.MessageError, 0, len(blkMsgs))
		blockMessageResults = messagemodel.BlockMessages{}
	)

	// Record which blocks had which messages, regardless of duplicates
	for _, bm := range blkMsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		blk := bm.Block
		for _, msg := range bm.SecpMessages {
			blockMessageResults = append(blockMessageResults, &messagemodel.BlockMessage{
				Height:  int64(bm.Block.Height),
				Block:   blk.Cid().String(),
				Message: msg.Cid().String(),
			})
		}
		for _, msg := range bm.BlsMessages {
			blockMessageResults = append(blockMessageResults, &messagemodel.BlockMessage{
				Height:  int64(bm.Block.Height),
				Block:   blk.Cid().String(),
				Message: msg.Cid().String(),
			})
		}
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		blockMessageResults,
	}, report, nil
}
