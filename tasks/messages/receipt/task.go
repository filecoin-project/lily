package receipt

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens/util"
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

	blkMsgRect, err := t.node.TipSetMessageReceipts(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting tipset message receipet: %w", err)
		return nil, report, nil
	}

	getActorCode, makeActorCodeRuncErr := util.MakeGetActorCodeFunc(ctx, t.node.Store(), current, executed)

	var (
		receiptResults = make(messagemodel.Receipts, 0, len(blkMsgRect))
		errorsDetected = make([]*messages.MessageError, 0, len(blkMsgRect))
		msgsSeen       = make(map[cid.Cid]bool, len(blkMsgRect))
	)

	for _, m := range blkMsgRect {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		itr, err := m.Iterator()
		if err != nil {
			return nil, nil, err
		}

		for itr.HasNext() {
			msg, index, rec := itr.Next()
			if msgsSeen[msg.Cid()] {
				continue
			}
			msgsSeen[msg.Cid()] = true

			rcpt := &messagemodel.Receipt{
				// use current's height and stateroot since receipts returned from TipSetMessageReceipts come from current
				// the messages from `executed` are applied (executed) to produce the stateroot and receipts in `current`.
				Height:    int64(current.Height()),
				StateRoot: current.ParentState().String(),

				Message:  msg.Cid().String(),
				Idx:      index,
				ExitCode: int64(rec.ExitCode),
				GasUsed:  rec.GasUsed,
				Return:   rec.Return,
			}
			toCode, found := getActorCode(ctx, msg.VMMessage().To)
			if found && rec.ExitCode.IsSuccess() && makeActorCodeRuncErr == nil {
				parsedReturn, _, err := util.ParseReturn(rec.Return, msg.VMMessage().Method, toCode)
				if err == nil {
					rcpt.ParsedReturn = parsedReturn
				}
			}
			receiptResults = append(receiptResults, rcpt)
		}

	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		receiptResults,
	}, report, nil
}
