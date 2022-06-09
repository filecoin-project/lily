package internal_message

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens"
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

func (p *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "internal_message"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	mex, err := p.node.MessageExecutions(ctx, current, executed)
	if err != nil {
		report.ErrorsDetected = fmt.Errorf("getting messages executions for tipset: %w", err)
		return nil, report, nil
	}

	var (
		internalResult = make(messagemodel.InternalMessageList, 0, len(mex))
		errorsDetected = make([]*messages.MessageError, 0) // we don't know the cap since mex is recursive in nature.
	)

	for _, parent := range mex {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		toName, toFamily, err := util.ActorNameAndFamilyFromCode(parent.ToActorCode)
		if err != nil {
			// TODO what do we do if there is an error? Continue with unknown family names or abort?
			errorsDetected = append(errorsDetected, &messages.MessageError{
				Cid:   parent.Cid,
				Error: fmt.Errorf("failed get message (%s) to actor name and family: %w", parent.Cid, err).Error(),
			})
		}
		if parent.Implicit {
			internalResult = append(internalResult, &messagemodel.InternalMessage{
				Height:        int64(parent.Height),
				Cid:           parent.Cid.String(),
				SourceMessage: "", // there is no source for implicit messages, they include cron tick and reward messages only
				StateRoot:     parent.StateRoot.String(),
				From:          parent.Message.From.String(),
				To:            parent.Message.To.String(),
				ActorName:     toName,
				ActorFamily:   toFamily,
				Value:         parent.Message.Value.String(),
				Method:        uint64(parent.Message.Method),
				ExitCode:      int64(parent.Ret.ExitCode),
				GasUsed:       parent.Ret.GasUsed,
			})
		} else {
			for _, child := range getChildMessagesOf(parent) {
				// Cid() computes a CID, so only call it once
				childCid := child.Message.Cid()
				childToName, childToFamily, err := util.ActorNameAndFamilyFromCode(childCid)
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   parent.Cid,
						Error: fmt.Errorf("failed get child message (%s) to actor name and family: %w", childCid, err).Error(),
					})
				}
				internalResult = append(internalResult, &messagemodel.InternalMessage{
					Height:        int64(parent.Height),
					Cid:           childCid.String(),
					StateRoot:     parent.StateRoot.String(),
					SourceMessage: parent.Cid.String(),
					From:          child.Message.From.String(),
					To:            child.Message.To.String(),
					Value:         child.Message.Value.String(),
					Method:        uint64(child.Message.Method),
					ActorName:     childToName,
					ActorFamily:   childToFamily,
					ExitCode:      int64(child.Receipt.ExitCode),
					GasUsed:       child.Receipt.GasUsed,
				})
			}
		}

	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}
	return internalResult, report, nil
}

func walkExecutionTrace(et *types.ExecutionTrace, trace *[]*MessageTrace) {
	for _, sub := range et.Subcalls {
		*trace = append(*trace, &MessageTrace{
			Message:   sub.Msg,
			Receipt:   sub.MsgRct,
			Error:     sub.Error,
			Duration:  sub.Duration,
			GasCharge: sub.GasCharges,
		})
		walkExecutionTrace(&sub, trace) //nolint:scopelint,gosec
	}
}

type MessageTrace struct {
	Message   *types.Message
	Receipt   *types.MessageReceipt
	Error     string
	Duration  time.Duration
	GasCharge []*types.GasTrace
}

func getChildMessagesOf(m *lens.MessageExecution) []*MessageTrace {
	var out []*MessageTrace
	walkExecutionTrace(&m.Ret.ExecutionTrace, &out)
	return out
}
