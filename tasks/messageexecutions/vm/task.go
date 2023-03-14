package vm

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
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

	var getActorCode func(ctx context.Context, a address.Address) (cid.Cid, bool)
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

		if parentMsg.Ret.ExitCode.IsError() {
			log.Debugf("skip parsing vm messages for source message %s with exit code %s", parentMsg.Cid, parentMsg.Ret.ExitCode.String())
			continue
		}

		// TODO this loop could be parallelized if it becomes a bottleneck.
		// NB: the getActorCode method is the expensive call since it resolves addresses and may load the statetree.
		for _, child := range util.GetChildMessagesOf(parentMsg) {
			// Cid() computes a CID, so only call it once
			childMsg := &types.Message{
				To:     child.Message.To,
				From:   child.Message.From,
				Value:  child.Message.Value,
				Method: child.Message.Method,
				Params: child.Message.Params,
				// these fields were deprecated in https://github.com/filecoin-project/lotus/commit/dbbcf4b2ee9626796e23a096c66e67ff350810e4
				Version:    0,
				GasLimit:   0,
				Nonce:      0,
				GasFeeCap:  abi.NewTokenAmount(0),
				GasPremium: abi.NewTokenAmount(0),
			}
			childCid := childMsg.Cid()

			toCode, found := getActorCode(ctx, child.Message.To)
			if !found && child.Receipt.ExitCode == 0 {
				// No destination actor code. Normally Lotus will create an account actor for unknown addresses but if the
				// message fails then Lotus will not allow the actor to be created, and we are left with an address of an
				// unknown type.
				// If the message was executed it means we are out of step with Lotus behaviour somehow. This probably
				// indicates that Lily actor type detection is out of date.
				errMsg := fmt.Sprintf("parsing VM message. source_cid %s, source_receipt %+v child_cid %s child_receipt %+v", parentMsg.Cid, parentMsg.Ret, childCid, child.Receipt)
				log.Error(errMsg)
				errorsDetected = append(errorsDetected, &messages.MessageError{
					Cid:   parentMsg.Cid,
					Error: fmt.Errorf("failed to get to actor code for message: %s to address %s: %s", childCid, child.Message.To, errMsg).Error(),
				})
				continue
			}

			toActorCode := "<Unknown>"
			if !toCode.Equals(cid.Undef) {
				toActorCode = toCode.String()
			}

			vmMsg := &messagemodel.VMMessage{
				Height:    int64(parentMsg.Height),
				StateRoot: parentMsg.StateRoot.String(),
				Source:    parentMsg.Cid.String(),
				Cid:       childCid.String(),
				From:      child.Message.From.String(),
				To:        child.Message.To.String(),
				Value:     child.Message.Value.String(),
				GasUsed:   0,
				ExitCode:  int64(child.Receipt.ExitCode),
				ActorCode: toActorCode,
				Method:    uint64(child.Message.Method),
				Index:     child.Index,
				// Params will be filled below if exit code is non-zero
				// Return will be filled below if exit code is non-zero
			}

			// only parse params and return of successful messages since unsuccessful messages don't return a parseable value.
			// As an example: a message may return ErrForbidden, it will have valid params, but will not contain a
			// parsable return value in its receipt.
			if child.Receipt.ExitCode.IsSuccess() {
				params, _, err := util.ParseVmMessageParams(child.Message.Params, child.Message.ParamsCodec, child.Message.Method, toCode)
				if err != nil {
					// a failure here indicates an error in message param parsing, or in exitcode checks above.
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid: parentMsg.Cid,
						// hex encode the params for reproduction in a unit test.
						Error: fmt.Errorf("failed parse child message params cid: %s to code: %s method: %d params (hex encoded): %s : %w",
							childCid, toCode, child.Message.Method, hex.EncodeToString(child.Message.Params), err).Error(),
					})
				} else {
					// add the message params
					if params != "" {
						vmMsg.Params = params
					}
				}

				ret, _, err := util.ParseVmMessageReturn(child.Receipt.Return, child.Receipt.ReturnCodec, child.Message.Method, toCode)
				if err != nil {
					errorsDetected = append(errorsDetected, &messages.MessageError{
						Cid: parentMsg.Cid,
						// hex encode the return for reproduction in a unit test.
						Error: fmt.Errorf("failed parse child message return cid: %s to code: %s method: %d return (hex encoded): %s : %w",
							childCid, toCode, child.Message.Method, hex.EncodeToString(child.Receipt.Return), err).Error(),
					})
				} else {
					// add the message return.
					if ret != "" {
						vmMsg.Returns = ret
					}
				}
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
