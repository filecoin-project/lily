package internal_parsed_message

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

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

var log = logging.Logger("lily/tasks/ipm")

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
			attribute.String("processor", "internal_parsed_message"),
		)
	}
	defer span.End()

	// execute in parallel as both operations are slow
	grp, _ := errgroup.WithContext(ctx)
	var mex []*lens.MessageExecution
	grp.Go(func() error {
		var err error
		mex, err = p.node.MessageExecutions(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages executions for tipset: %w", err)
		}
		return nil
	})

	var getActorCode func(a address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCode, err = util.MakeGetActorCodeFunc(ctx, p.node.Store(), current, executed)
		if err != nil {
			return fmt.Errorf("failed to make actor code query function: %w", err)
		}
		return nil
	})

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	// if either fail, report error and bail
	if err := grp.Wait(); err != nil {
		report.ErrorsDetected = err
		return nil, report, nil
	}

	var (
		internalParsedResult = make(messagemodel.InternalParsedMessageList, 0, len(mex))
		errorsDetected       = make([]*messages.MessageError, 0) // we don't know the cap since mex is recursive in nature.
	)

	for _, parent := range mex {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		if parent.Implicit {
			method, params, err := util.MethodAndParamsForMessage(parent.Message, parent.ToActorCode)
			if err != nil {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   parent.Cid,
					Error: fmt.Errorf("failed parse method and params for message: %w", err).Error(),
				})
			}
			internalParsedResult = append(internalParsedResult, &messagemodel.InternalParsedMessage{
				Height: int64(parent.Height),
				Cid:    parent.Cid.String(),
				From:   parent.Message.From.String(),
				To:     parent.Message.To.String(),
				Value:  parent.Message.Value.String(),
				Method: method,
				Params: params,
			})
		} else {
			for _, child := range getChildMessagesOf(parent) {
				// Cid() computes a CID, so only call it once
				childCid := child.Message.Cid()
				toCode, ok := getActorCode(child.Message.To)
				if !ok {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   parent.Cid,
						Error: fmt.Errorf("failed to get to actor code for message: %s", childCid).Error(),
					})
				}
				method, params, err := util.MethodAndParamsForMessage(child.Message, toCode)
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   parent.Cid,
						Error: fmt.Errorf("failed get child message (%s) to actor name and family: %w", childCid, err).Error(),
					})
				}
				internalParsedResult = append(internalParsedResult, &messagemodel.InternalParsedMessage{
					Height: int64(parent.Height),
					Cid:    childCid.String(),
					From:   child.Message.From.String(),
					To:     child.Message.To.String(),
					Value:  child.Message.Value.String(),
					Method: method,
					Params: params,
				})
			}
		}
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}
	return internalParsedResult, report, nil
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
