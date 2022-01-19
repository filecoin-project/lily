package messageexecutions

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/taskapi"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/messages"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"
)

func NewTask(node taskapi.TaskAPI) *Task {
	return &Task{
		node: node,
	}
}

type Task struct {
	node taskapi.TaskAPI
}

func (p *Task) ProcessTipSets(ctx context.Context, child, parent *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error) {
	mex, err := p.node.GetMessageExecutionsForTipSet(ctx, child, parent)
	if err != nil {
		return nil, nil, err
	}
	data, report, err := p.ProcessMessageExecutions(ctx, p.node.Store(), child, parent, mex)
	if err != nil {
		return nil, nil, err
	}
	return data, visormodel.ProcessingReportList{report}, err
}

func (p *Task) Close() error {
	return nil
}

func (p *Task) ProcessMessageExecutions(ctx context.Context, store adt.Store, ts *types.TipSet, pts *types.TipSet, mex []*lens.MessageExecution) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessMessageExecutions")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.String()), attribute.Int64("height", int64(ts.Height())))
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(pts.Height()),
		StateRoot: pts.ParentState().String(),
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

		// TODO(frrist): this code is commented out as it collects all internal message sent through the VM.
		// Currently there does not exist a need for message analysis at this granularity.
		// Before enabling this, some type of filtering will need to be implemented such that only
		// the internal sends we are interested in can be extracted.
		/*
			getActorCode, err := util.MakeGetActorCodeFunc(ctx, store, ts, pts)
			if err != nil {
				return nil, nil, err
			}

			// some messages will cause internal messages to be sent between the actors, gather and record them here.
			subCalls := getChildMessagesOf(m)
			for _, sub := range subCalls {
				subToActorCode, found := getActorCode(sub.Message.To)
				var subToName, subToFamily string
				if found {
					subToName, subToFamily, err = util.ActorNameAndFamilyFromCode(subToActorCode)
					if err != nil {
						errorsDetected = append(errorsDetected, &messages.MessageError{
							Cid:   sub.Message.Cid(),
							Error: xerrors.Errorf("failed to get sub-message to actor name and family: %w", err).Error(),
						})
					}
					// if the message aborted execution while creating an actor due to lack of gas then this is expected as the actors don't exist
				} else {
					subToName = "<unknown>"
					subToFamily = "<unknown>"
				}
				internalResult = append(internalResult, &messagemodel.InternalMessage{
					Height:        int64(m.Height),
					Cid:           sub.Message.Cid().String(), // pity we have to calculate this here
					StateRoot:     m.StateRoot.String(),       // state root of the parent message
					SourceMessage: m.Cid.String(),
					From:          sub.Message.From.String(),
					To:            sub.Message.To.String(),
					Value:         sub.Message.Value.String(),
					Method:        uint64(sub.Message.Method),
					ActorName:     subToName,
					ActorFamily:   subToFamily,
					ExitCode:      int64(sub.Receipt.ExitCode),
					GasUsed:       sub.Receipt.GasUsed,
				})

				subMethod, subParams, err := util.MethodAndParamsForMessage(sub.Message, subToActorCode)
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   sub.Message.Cid(),
						Error: xerrors.Errorf("failed parse method and params for sub-message: %w", err).Error(),
					})
				}
				internalParsedResult = append(internalParsedResult, &messagemodel.InternalParsedMessage{
					Height: int64(m.Height),
					Cid:    m.Cid.String(),
					From:   m.Message.From.String(),
					To:     m.Message.To.String(),
					Value:  m.Message.Value.String(),
					Method: subMethod,
					Params: subParams,
				})
			}
		*/
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
