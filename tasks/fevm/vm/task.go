package fevmvm

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	builtintypes "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/fevm"
	visormodel "github.com/filecoin-project/lily/model/visor"
	tasks "github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/tasks/fevmvm")

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
		// these fields were deprecated
		// in https://github.com/filecoin-project/lotus/commit/dbbcf4b2ee9626796e23a096c66e67ff350810e4
		Version:    0,
		GasLimit:   0,
		Nonce:      0,
		GasFeeCap:  abi.NewTokenAmount(0),
		GasPremium: abi.NewTokenAmount(0),
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

func (t *Task) ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error) {
	ctx, span := otel.Tracer("").Start(ctx, "ProcessTipSets")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("processor", "fevm_vm_messages"),
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
		vmMessageResults = make(fevm.FEVMVMMessageList, 0)
	)

	errs := []error{}

	for _, parentMsg := range mex {
		// Only handle EVM related message
		if !util.IsEVMAddress(ctx, t.node, parentMsg.Message.From, current.Key()) && !util.IsEVMAddress(ctx, t.node, parentMsg.Message.To, current.Key()) && !(parentMsg.Message.To != builtintypes.EthereumAddressManagerActorAddr) {
			continue
		}
		messageHash, err := ethtypes.EthHashFromCid(parentMsg.Cid)
		if err != nil {
			log.Errorf("Error at finding hash: [cid: %v] err: %v", parentMsg.Cid, err)
			errs = append(errs, err)
			continue
		}
		transaction, err := t.node.EthGetTransactionByHash(ctx, &messageHash)
		if err != nil {
			log.Errorf("Error at getting receipt: [hash: %v] err: %v", messageHash, err)
			errs = append(errs, err)
			continue
		}

		if transaction == nil {
			continue
		}

		log.Infof("message: %v, %v", parentMsg.Cid, base64.StdEncoding.EncodeToString(parentMsg.Message.Params))

		for _, child := range util.GetChildMessagesOf(parentMsg) {
			toCode, _ := getActorCode(ctx, child.Message.To)

			toActorCode := "<Unknown>"
			if !toCode.Equals(cid.Undef) {
				toActorCode = toCode.String()
			}
			fromEthAddress := getEthAddress(child.Message.From)
			toEthAddress := getEthAddress(child.Message.To)

			vmMsg := &fevm.FEVMVMMessage{
				Height:          int64(parentMsg.Height),
				TransactionHash: transaction.Hash.String(),
				StateRoot:       parentMsg.StateRoot.String(),
				Source:          parentMsg.Cid.String(),
				Cid:             getMessageTraceCid(child.Message).String(),
				To:              child.Message.To.String(),
				From:            child.Message.From.String(),
				FromEthAddress:  fromEthAddress,
				ToEthAddress:    toEthAddress,
				Value:           child.Message.Value.String(),
				GasUsed:         0,
				ExitCode:        int64(child.Receipt.ExitCode),
				ActorCode:       toActorCode,
				Method:          uint64(child.Message.Method),
				Index:           child.Index,
				Params:          base64.StdEncoding.EncodeToString(child.Message.Params),
				Returns:         base64.StdEncoding.EncodeToString(child.Receipt.Return),
			}

			// only parse params and return of successful messages since unsuccessful messages don't return a parseable value.
			// As an example: a message may return ErrForbidden, it will have valid params, but will not contain a
			// parsable return value in its receipt.
			if child.Receipt.ExitCode.IsSuccess() {
				params, _, err := util.ParseVmMessageParams(child.Message.Params, child.Message.ParamsCodec, child.Message.Method, toCode)
				if err == nil {
					vmMsg.ParsedParams = params
				}
				ret, _, err := util.ParseVmMessageReturn(child.Receipt.Return, child.Receipt.ReturnCodec, child.Message.Method, toCode)
				if err == nil {
					vmMsg.ParsedReturns = ret
				}
			}

			// append message to results
			vmMessageResults = append(vmMessageResults, vmMsg)
		}
	}

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	} else {
		err = nil
	}

	return vmMessageResults, report, err
}
