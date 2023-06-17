package fevmtransaction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model/fevm"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

var log = logging.Logger("lily/tasks/fevmtransaction")

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
			attribute.String("processor", "fevm_transaction"),
		)
	}
	defer span.End()

	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
	}

	messages, err := p.node.ChainGetMessagesInTipset(ctx, current.Key())
	if err != nil {
		log.Errorf("Error at getting messages. ts: %v, height: %v, err: %v", current.String(), current.Height(), err)
		report.ErrorsDetected = err
		return nil, report, nil
	}
	errs := []error{}
	out := make(fevm.FEVMTransactionList, 0)
	for _, message := range messages {
		if message.Message == nil {
			continue
		}
		if !util.IsEVMMessage(ctx, p.node, message.Message, current.Key()) {
			continue
		}

		hash, err := ethtypes.EthHashFromCid(message.Cid)
		if err != nil {
			log.Errorf("Error at finding hash: [cid: %v] err: %v", message.Cid, err)
			errs = append(errs, err)
			continue
		}

		txn, err := p.node.EthGetTransactionByHash(ctx, &hash)
		if err != nil {
			log.Errorf("Error at getting transaction: [hash: %v] err: %v", hash, err)
			errs = append(errs, err)
			continue
		}

		if txn == nil {
			continue
		}

		txnObj := &fevm.FEVMTransaction{
			Height:               int64(current.Height()),
			Hash:                 txn.Hash.String(),
			ChainID:              uint64(txn.ChainID),
			Nonce:                uint64(txn.Nonce),
			From:                 txn.From.String(),
			Value:                txn.Value.Int.String(),
			Type:                 uint64(txn.Type),
			Input:                txn.Input.String(),
			Gas:                  uint64(txn.Gas),
			MaxFeePerGas:         txn.MaxFeePerGas.Int.String(),
			MaxPriorityFeePerGas: txn.MaxPriorityFeePerGas.Int.String(),
			V:                    txn.V.String(),
			R:                    txn.R.String(),
			S:                    txn.S.String(),
		}

		if txn.BlockHash != nil {
			txnObj.BlockHash = txn.BlockHash.String()
		}
		if txn.BlockNumber != nil {
			txnObj.BlockNumber = uint64(*txn.BlockNumber)
		}
		if txn.TransactionIndex != nil {
			txnObj.TransactionIndex = uint64(*txn.TransactionIndex)
		}
		if txn.To != nil {
			txnObj.To = txn.To.String()
		}

		if len(txn.AccessList) > 0 {
			accessStrList := make([]string, 0)
			for _, access := range txn.AccessList {
				accessStrList = append(accessStrList, access.String())
			}
			b, err := json.Marshal(accessStrList)
			if err == nil {
				txnObj.AccessList = string(b)
			}
		}
		out = append(out, txnObj)
	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return model.PersistableList{out}, report, nil
}
