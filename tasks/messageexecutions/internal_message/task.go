package internal_message

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
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

	for _, m := range mex {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		// we don't yet record implicit messages in the other message task, record them here.
		if m.Implicit {
			toName, toFamily, err := util.ActorNameAndFamilyFromCode(m.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: fmt.Errorf("failed get message to actor name and family: %w", err).Error(),
				})
			}
			internalResult = append(internalResult, &messagemodel.InternalMessage{
				Height:        int64(m.Height),
				Cid:           m.Cid.String(),
				SourceMessage: "", // there is no source for implicit messages, they include cron tick and reward messages only
				StateRoot:     m.StateRoot.String(),
				From:          m.Message.From.String(),
				To:            m.Message.To.String(),
				ActorName:     toName,
				ActorFamily:   toFamily,
				Value:         m.Message.Value.String(),
				Method:        uint64(m.Message.Method),
				ExitCode:      int64(m.Ret.ExitCode),
				GasUsed:       m.Ret.GasUsed,
			})
		}

	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}
	return internalResult, report, nil
}
