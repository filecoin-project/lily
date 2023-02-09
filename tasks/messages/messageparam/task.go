package messageparam

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

func (t *Task) ProcessTipSet(ctx context.Context, current *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("processor", "messages"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	blksMsgs, err := t.node.TipSetBlockMessages(ctx, current)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting messages for tipset: %w", err)
		return nil, report, nil
	}

	var (
		messageResults = make(messagemodel.MessageParamList, 0)
		errorsDetected = make([]*messages.MessageError, 0)
		blkMsgSeen     = make(map[cid.Cid]bool)
	)

	// record all unique messages in current
	for _, blkMsgs := range blksMsgs {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}
		for _, msg := range blkMsgs.BlsMessages {
			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			// record all unique messages with params
			if len(msg.Params) == 0 {
				continue
			}
			messageResults = append(messageResults, &messagemodel.MessageParam{
				Cid:    msg.Cid().String(),
				Params: msg.Params,
			})
		}
		for _, msg := range blkMsgs.SecpMessages {
			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			if len(msg.Message.Params) == 0 {
				continue
			}

			// record all unique messages
			messageResults = append(messageResults, &messagemodel.MessageParam{
				Cid:    msg.Cid().String(),
				Params: msg.Message.Params,
			})

		}

	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		messageResults,
	}, report, nil
}
