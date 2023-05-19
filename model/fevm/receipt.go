package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMReceipt struct {
	tableName struct{} `pg:"fevm_receipt"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`

	// Message CID
	Message string `pg:",use_zero"`

	TransactionHash   string `pg:",notnull"`
	TransactionIndex  uint64 `pg:",use_zero"`
	BlockHash         string `pg:",notnull"`
	BlockNumber       uint64 `pg:",use_zero"`
	From              string `pg:",notnull"`
	To                string `pg:",notnull"`
	ContractAddress   string `pg:",notnull"`
	Status            uint64 `pg:",use_zero"`
	CumulativeGasUsed uint64 `pg:",use_zero"`
	GasUsed           uint64 `pg:",use_zero"`
	EffectiveGasPrice int64  `pg:",use_zero"`
	LogsBloom         string `pg:",notnull"`
	Logs              string `pg:",type:jsonb"`
}

func (f *FEVMReceipt) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_receipt"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMReceiptList []*FEVMReceipt

func (f FEVMReceiptList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_receipt"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
