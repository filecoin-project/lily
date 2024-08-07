package fevmtrace

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/fevm"
	visormodel "github.com/filecoin-project/lily/model/visor"
	tasks "github.com/filecoin-project/lily/tasks"

	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

var log = logging.Logger("lily/tasks/fevmtrace")

type Task struct {
	node tasks.DataSource
}

func NewTask(node tasks.DataSource) *Task {
	return &Task{node: node}
}

func getMessageTraceCid(message types.MessageTrace) cid.Cid {
	childMsg := &types.Message{
		To:     message.To,
		From:   message.From,
		Value:  message.Value,
		Method: message.Method,
		Params: message.Params,
	}

	return childMsg.Cid()
}

func getEthAddress(addr address.Address) string {
	to, err := ethtypes.EthAddressFromFilecoinAddress(addr)
	if err != nil {
		log.Warnf("Error at getting eth address: [message address: %v] err: %v", addr.String(), err)
		return ""
	}

	return to.String()
}

func (t *Task) getActorAddress(ctx context.Context, address address.Address, tsk types.TipSetKey) address.Address {
	actor, err := t.node.Actor(ctx, address, tsk)
	if err == nil && actor != nil && actor.DelegatedAddress != nil {
		return *actor.DelegatedAddress
	}
	return address
}

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "fevm_trace"),
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
		traceResults = make(fevm.FEVMTraceList, 0)
	)

	errs := []error{}

	for _, parentMsg := range mex {
		// Only handle EVM related message
		if !util.IsEVMMessage(ctx, t.node, parentMsg.Message, current.Key()) {
			continue
		}
		messageHash, err := ethtypes.EthHashFromCid(parentMsg.Cid)
		if err != nil {
			log.Errorf("Error at finding hash: [cid: %v] err: %v", parentMsg.Cid, err)
			errs = append(errs, err)
			continue
		}
		txn, err := t.node.EthGetTransactionByHash(ctx, &messageHash)
		if err != nil {
			log.Errorf("Error at getting transaction: [hash: %v] err: %v", messageHash, err)
			errs = append(errs, err)
			continue
		}

		if txn == nil {
			log.Errorf("transaction: [hash: %v] is null", messageHash)
			continue
		}
		transactionHash := txn.Hash.String()

		for _, child := range util.GetChildMessagesOf(parentMsg) {
			fromCode, _ := getActorCode(ctx, child.Message.From)
			var fromActorCode string
			if !fromCode.Equals(cid.Undef) {
				fromActorCode, _, err = util.ActorNameAndFamilyFromCode(fromCode)
				if err != nil {
					errs = append(errs, err)
				}
			}

			toCode, _ := getActorCode(ctx, child.Message.To)
			actorCode := "<Unknown>"
			var toActorCode string
			if !toCode.Equals(cid.Undef) {
				actorCode = toCode.String()
				toActorCode, _, err = util.ActorNameAndFamilyFromCode(toCode)
				if err != nil {
					errs = append(errs, err)
				}
			}

			// Get Actor Address
			toAddress := t.getActorAddress(ctx, child.Message.To, current.Key())
			fromAddress := t.getActorAddress(ctx, child.Message.From, current.Key())

			traceObj := &fevm.FEVMTrace{
				Height:              int64(parentMsg.Height),
				TransactionHash:     transactionHash,
				MessageStateRoot:    parentMsg.StateRoot.String(),
				MessageCid:          parentMsg.Cid.String(),
				TraceCid:            getMessageTraceCid(child.Message).String(),
				FromFilecoinAddress: fromAddress.String(),
				ToFilecoinAddress:   toAddress.String(),
				From:                getEthAddress(fromAddress),
				To:                  getEthAddress(toAddress),
				Value:               child.Message.Value.String(),
				ExitCode:            int64(child.Receipt.ExitCode),
				ActorCode:           actorCode,
				Method:              uint64(child.Message.Method),
				Index:               child.Index,
				Params:              ethtypes.EthBytes(child.Message.Params).String(),
				Returns:             ethtypes.EthBytes(child.Receipt.Return).String(),
				ParamsCodec:         child.Message.ParamsCodec,
				ReturnsCodec:        child.Receipt.ReturnCodec,
				ToActorName:         toActorCode,
				FromActorName:       fromActorCode,
			}

			// only parse params and return of successful messages since unsuccessful messages don't return a parseable value.
			// As an example: a message may return ErrForbidden, it will have valid params, but will not contain a
			// parsable return value in its receipt.
			if child.Receipt.ExitCode.IsSuccess() {
				params, parsedMethod, err := util.ParseVmMessageParams(child.Message.Params, child.Message.ParamsCodec, child.Message.Method, toCode)
				// in ParseVmMessageParams it will return actor name when actor not found
				if err == nil && parsedMethod != builtin.ActorNameByCode(toCode) {
					traceObj.ParsedParams = params
					traceObj.ParsedMethod = parsedMethod
				}
				ret, parsedMethod, err := util.ParseVmMessageReturn(child.Receipt.Return, child.Receipt.ReturnCodec, child.Message.Method, toCode)
				// in ParseVmMessageParams it will return actor name when actor not found
				if err == nil && parsedMethod != builtin.ActorNameByCode(toCode) {
					traceObj.ParsedReturns = ret
				}
			}

			// append message to results
			traceResults = append(traceResults, traceObj)
		}
	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return traceResults, report, nil
}
