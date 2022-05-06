package message

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
			attribute.String("processor", "messages"),
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
	blkMsgs := tsMsgs.Block

	var (
		messageResults = make(messagemodel.Messages, 0, len(blkMsgs))
		errorsDetected = make([]*messages.MessageError, 0, len(blkMsgs))
		blkMsgSeen     = make(map[cid.Cid]bool)
	)

	// Record which blocks had which messages, regardless of duplicates
	for _, bm := range blkMsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		for _, msg := range bm.SecpMessages {
			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			var msgSize int
			if b, err := msg.Message.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   msg.Cid(),
					Error: fmt.Errorf("failed to serialize message: %w", err).Error(),
				})
			}

			// record all unique Secp messages
			msg := &messagemodel.Message{
				Height:     int64(bm.Block.Height),
				Cid:        msg.Cid().String(),
				From:       msg.Message.From.String(),
				To:         msg.Message.To.String(),
				Value:      msg.Message.Value.String(),
				GasFeeCap:  msg.Message.GasFeeCap.String(),
				GasPremium: msg.Message.GasPremium.String(),
				GasLimit:   msg.Message.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      msg.Message.Nonce,
				Method:     uint64(msg.Message.Method),
			}
			messageResults = append(messageResults, msg)

		}
		for _, msg := range bm.BlsMessages {
			if blkMsgSeen[msg.Cid()] {
				continue
			}
			blkMsgSeen[msg.Cid()] = true

			var msgSize int
			if b, err := msg.Serialize(); err == nil {
				msgSize = len(b)
			} else {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   msg.Cid(),
					Error: fmt.Errorf("failed to serialize message: %w", err).Error(),
				})
			}

			// record all unique bls messages
			msg := &messagemodel.Message{
				Height:     int64(bm.Block.Height),
				Cid:        msg.Cid().String(),
				From:       msg.From.String(),
				To:         msg.To.String(),
				Value:      msg.Value.String(),
				GasFeeCap:  msg.GasFeeCap.String(),
				GasPremium: msg.GasPremium.String(),
				GasLimit:   msg.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      msg.Nonce,
				Method:     uint64(msg.Method),
			}
			messageResults = append(messageResults, msg)
		}
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		messageResults,
	}, report, nil
}
