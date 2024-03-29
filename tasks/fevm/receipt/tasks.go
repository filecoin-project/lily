package fevmreceipt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/fevm"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

var log = logging.Logger("lily/tasks/fevmreceipt")

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
			attribute.String("processor", "fevm_receipt"),
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
	out := make(fevm.FEVMReceiptList, 0)
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

		receipt, err := p.node.EthGetTransactionReceipt(ctx, hash)
		if err != nil {
			log.Errorf("Error at getting receipt: [hash: %v] err: %v", hash, err)
			errs = append(errs, err)
			continue
		}

		if receipt == nil {
			continue
		}

		receiptObj := &fevm.FEVMReceipt{
			Height:            int64(current.Height()),
			TransactionHash:   receipt.TransactionHash.String(),
			TransactionIndex:  uint64(receipt.TransactionIndex),
			BlockHash:         receipt.BlockHash.String(),
			BlockNumber:       uint64(receipt.BlockNumber),
			From:              receipt.From.String(),
			Status:            uint64(receipt.Status),
			CumulativeGasUsed: uint64(receipt.CumulativeGasUsed),
			GasUsed:           uint64(receipt.GasUsed),
			EffectiveGasPrice: receipt.EffectiveGasPrice.Int64(),
			LogsBloom:         hex.EncodeToString(receipt.LogsBloom),
			Message:           message.Cid.String(),
		}

		b, err := json.Marshal(receipt.Logs)
		if err == nil {
			receiptObj.Logs = string(b)
		}
		if receipt.ContractAddress != nil {
			receiptObj.ContractAddress = receipt.ContractAddress.String()
		}
		if receipt.To != nil {
			receiptObj.To = receipt.To.String()
		}
		out = append(out, receiptObj)

	}

	if len(errs) > 0 {
		report.ErrorsDetected = fmt.Errorf("%v", errs)
	}

	return model.PersistableList{out}, report, nil
}
