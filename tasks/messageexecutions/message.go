package messageexecutions

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/messages"
)

type Task struct {
	node task.TaskAPI
}

func NewTask(node task.TaskAPI) *Task {
	return &Task{
		node: node,
	}
}

func (p *Task) ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessMessageExecutions")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
	}

	mex, err := p.node.GetMessageExecutionsForTipSet(ctx, ts, pts)
	if err != nil {
		report.ErrorsDetected = xerrors.Errorf("getting messages executions for tipset: %w", err)
		return nil, report, nil
	}

	var (
		internalResult       = make(messagemodel.InternalMessageList, 0, len(mex))
		internalParsedResult = make(messagemodel.InternalParsedMessageList, 0, len(mex))
		errorsDetected       = make([]*messages.MessageError, 0) // we don't know the cap since mex is recursive in nature.
	)

	for _, m := range mex {
		select {
		case <-ctx.Done():
			return nil, nil, xerrors.Errorf("context done: %w", ctx.Err())
		default:
		}

		// we don't yet record implicit messages in the other message task, record them here.
		if m.Implicit {
			toName, toFamily, err := util.ActorNameAndFamilyFromCode(m.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: xerrors.Errorf("failed get message to actor name and family: %w", err).Error(),
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
			method, params, err := util.MethodAndParamsForMessage(m.Message, m.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: xerrors.Errorf("failed parse method and params for message: %w", err).Error(),
				})
			}
			internalParsedResult = append(internalParsedResult, &messagemodel.InternalParsedMessage{
				Height: int64(m.Height),
				Cid:    m.Cid.String(),
				From:   m.Message.From.String(),
				To:     m.Message.To.String(),
				Value:  m.Message.Value.String(),
				Method: method,
				Params: params,
			})
		}

	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}
	return model.PersistableList{
		internalResult,
		internalParsedResult,
	}, report, nil
}

/*
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
*/
