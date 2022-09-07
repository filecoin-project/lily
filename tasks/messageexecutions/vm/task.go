package vm

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/exitcode"
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
	tasks "github.com/filecoin-project/lily/tasks"
	messages "github.com/filecoin-project/lily/tasks/messages"
)

var log = logging.Logger("lily/tasks/vmmsg")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{node: node}
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "vm_messages"),
		)
	}
	defer span.End()

	// execute in parallel as both operations are slow
	grp, _ := errgroup.WithContext(ctx)
	var mex []*lens.MessageExecution
	grp.Go(func() error {
		var err error
		mex, err = t.node.MessageExecutions(ctx, current, executed)
		if err != nil {
			return fmt.Errorf("getting messages executions for tipset: %w", err)
		}
		return nil
	})

	var getActorCode func(a address.Address) (cid.Cid, bool)
	grp.Go(func() error {
		var err error
		getActorCode, err = util.MakeGetActorCodeFunc(ctx, t.node.Store(), current, executed)
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
		vmMessageResults = make(messagemodel.VMMessageList, 0, len(mex))
		errorsDetected   = make([]*messages.MessageError, 0)
	)
	for _, parentMsg := range mex {
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("context done: %w", ctx.Err())
		default:
		}

		// TODO this loop could be parallelized if it becomes a bottleneck.
		// NB: the getActorCode method is the expensive call since it resolves addresses and may load the statetree.
		for _, child := range util.GetChildMessagesOf(parentMsg) {
			// Cid() computes a CID, so only call it once
			childCid := child.Message.Cid()

			toCode, found := getActorCode(child.Message.To)
			if !found && child.Receipt.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created, and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				log.Errorw("parsing VM message", "source_cid", parentMsg.Cid, "source_receipt", parentMsg.Ret, "child_cid", childCid, "child_receipt", child.Receipt)
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   parentMsg.Cid,
					Error: fmt.Errorf("failed to get to actor code for message: %s to address %s", childCid, child.Message.To).Error(),
				})
				continue
			}

			// if the to actor code was not found we cannot parse params or return, record the message and continue
			if !found ||
				// if the exit code indicates an issue with params or method we cannot parse the message params
				child.Receipt.ExitCode == exitcode.ErrSerialization ||
				child.Receipt.ExitCode == exitcode.ErrIllegalArgument ||
				child.Receipt.ExitCode == exitcode.SysErrInvalidMethod ||
				// UsrErrUnsupportedMethod TODO: https://github.com/filecoin-project/go-state-types/pull/44
				child.Receipt.ExitCode == exitcode.ExitCode(22) {

				// append results and continue
				vmMessageResults = append(vmMessageResults, &messagemodel.VMMessage{
					Height:    int64(parentMsg.Height),
					StateRoot: parentMsg.StateRoot.String(),
					Source:    parentMsg.Cid.String(),
					Cid:       childCid.String(),
					From:      child.Message.From.String(),
					To:        child.Message.To.String(),
					Value:     child.Message.Value.String(),
					GasUsed:   child.Receipt.GasUsed,
					ExitCode:  int64(child.Receipt.ExitCode), // exit code is guaranteed to be non-zero which will indicate why actor was not found (i.e. message that created the actor failed to apply)
					ActorCode: toCode.String(),               // since the actor code wasn't found this will be the string of an undefined CID.
					Method:    uint64(child.Message.Method),
					Params:    "",
					Returns:   "",
				})
				continue
			}

			// the to actor code was found and its exit code indicates the params should be parsable. We can safely
			// attempt to parse message params and return, but exit code may still be non-zero here.

			params, _, err := util.ParseParams(child.Message.Params, child.Message.Method, toCode)
			if err != nil {
				// a failure here indicates an error in message param parsing, or in exitcode checks above.
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid: parentMsg.Cid,
					// hex encode the params for reproduction in a unit test.
					Error: fmt.Errorf("failed parse child message params cid: %s to code: %s method: %d params (hex encoded): %s : %w",
						childCid, toCode, child.Message.Method, hex.EncodeToString(child.Message.Params), err).Error(),
				})
				// don't append message to result as it may contain invalud data.
				continue
			}

			// params successfully parsed.
			vmMsg := &messagemodel.VMMessage{
				Height:    int64(parentMsg.Height),
				StateRoot: parentMsg.StateRoot.String(),
				Source:    parentMsg.Cid.String(),
				Cid:       childCid.String(),
				From:      child.Message.From.String(),
				To:        child.Message.To.String(),
				Value:     child.Message.Value.String(),
				GasUsed:   child.Receipt.GasUsed,
				ExitCode:  int64(child.Receipt.ExitCode),
				ActorCode: toCode.String(),
				Method:    uint64(child.Message.Method),
				Params:    params,
				// Return will be filled below if exit code is non-zero
			}

			// only parse return of successful messages since unsuccessful messages don't return a parseable value.
			// As an example: a message may return ErrForbidden, it will have valid params, but will not contain a
			// parsable return value in its receipt.
			if child.Receipt.ExitCode.IsSuccess() {
				ret, _, err := util.ParseReturn(child.Receipt.Return, child.Message.Method, toCode)
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid: parentMsg.Cid,
						// hex encode the return for reproduction in a unit test.
						Error: fmt.Errorf("failed parse child message return cid: %s to code: %s method: %d return (hex encoded): %s : %w",
							childCid, toCode, child.Message.Method, hex.EncodeToString(child.Receipt.Return), err).Error(),
					})
					// don't append message to result as it may contain invalid data.
					continue
				}
				// add the message return.
				vmMsg.Returns = ret
			}
			// append message to results
			vmMessageResults = append(vmMessageResults, vmMsg)
		}
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}
	return vmMessageResults, report, nil
}
