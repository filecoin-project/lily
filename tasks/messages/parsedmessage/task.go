package parsedmessage

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/exitcode"
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
			attribute.String("processor", "parsed_messages"),
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
		parsedMessageResults = make(messagemodel.ParsedMessages, 0, len(emsgs))
		errorsDetected       = make([]*messages.MessageError, 0, len(emsgs))
		exeMsgSeen           = make(map[cid.Cid]bool, len(emsgs))
		totalGasLimit        int64
		totalUniqGasLimit    int64
	)

	for _, m := range emsgs {
		// Stop processing if we have been told to cancel
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		// calculate total gas limit of executed messages regardless of duplicates.
		for range m.Blocks {
			totalGasLimit += m.Message.GasLimit
		}

		if exeMsgSeen[m.Cid] {
			continue
		}
		exeMsgSeen[m.Cid] = true
		totalUniqGasLimit += m.Message.GasLimit

		if m.ToActorCode.Defined() {
			method, params, err := util.MethodAndParamsForMessage(m.Message, m.ToActorCode)
			if err == nil {
				pm := &messagemodel.ParsedMessage{
					Height: int64(m.Height),
					Cid:    m.Cid.String(),
					From:   m.Message.From.String(),
					To:     m.Message.To.String(),
					Value:  m.Message.Value.String(),
					Method: method,
					Params: params,
				}
				parsedMessageResults = append(parsedMessageResults, pm)
			} else {
				if m.Receipt.ExitCode == exitcode.ErrSerialization || m.Receipt.ExitCode == exitcode.ErrIllegalArgument || m.Receipt.ExitCode == exitcode.SysErrInvalidMethod {
					// ignore the parse error since the params are probably malformed, as reported by the vm
				} else {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid:   m.Cid,
						Error: fmt.Errorf("failed to parse message params: %w", err).Error(),
					})
				}
			}
		} else {
			// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
			// message fails then Lotus will not allow the actor to be created and we are left with an address of an
			// unknown type.
			// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
			// indicates that Lily actor type detection is out of date.
			if m.Receipt.ExitCode == 0 {
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   m.Cid,
					Error: fmt.Errorf("failed to parse message params: missing to actor code").Error(),
				})
			}
		}
	}
	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return model.PersistableList{
		parsedMessageResults,
	}, report, nil
}
