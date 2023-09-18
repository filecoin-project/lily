package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMReceipt struct {
	tableName struct{} `pg:"fevm_receipts"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Message CID
	Message string `pg:",use_zero"`
	// Hash of transaction.
	TransactionHash string `pg:",notnull"`
	// Integer of the transactions index position in the block.
	TransactionIndex uint64 `pg:",use_zero"`
	// Hash of the block where this transaction was in.
	BlockHash string `pg:",notnull"`
	// Block number where this transaction was in.
	BlockNumber uint64 `pg:",use_zero"`
	// ETH Address of the sender.
	From string `pg:",notnull"`
	// ETH Address of the receiver.
	To string `pg:",notnull"`
	// The contract address created, if the transaction was a contract creation, otherwise null.
	ContractAddress string `pg:",notnull"`
	// 0 indicates transaction failure , 1 indicates transaction succeeded.
	Status uint64 `pg:",use_zero"`
	// The total amount of gas used when this transaction was executed in the block.
	CumulativeGasUsed uint64 `pg:",use_zero"`
	// The actual amount of gas used in this block.
	GasUsed uint64 `pg:",use_zero"`
	// The actual value per gas deducted from the senders account.
	EffectiveGasPrice int64 `pg:",use_zero"`
	// Includes the bloom filter representation of the logs
	LogsBloom string `pg:",notnull"`
	// Array of log objects, which this transaction generated.
	Logs string `pg:",type:jsonb"`
}

func (f *FEVMReceipt) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_receipts"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMReceiptList []*FEVMReceipt

func (f FEVMReceiptList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_receipts"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
