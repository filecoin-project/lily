package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMTransaction struct {
	tableName struct{} `pg:"fevm_transactions"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`

	Hash                 string `pg:",pk,notnull"`
	ChainID              uint64 `pg:",use_zero"`
	Nonce                uint64 `pg:",use_zero"`
	BlockHash            string `pg:",notnull"`
	BlockNumber          uint64 `pg:",use_zero"`
	TransactionIndex     uint64 `pg:",use_zero"`
	From                 string `pg:",notnull"`
	To                   string `pg:",notnull"`
	Value                string `pg:",notnull"`
	Type                 uint64 `pg:",use_zero"`
	Input                string `pg:",notnull"`
	ParsedInput          string `pg:",type:jsonb"`
	Gas                  uint64 `pg:",use_zero"`
	MaxFeePerGas         string `pg:"type:numeric,notnull"`
	MaxPriorityFeePerGas string `pg:"type:numeric,notnull"`
	AccessList           string `pg:",type:jsonb"`
	V                    string `pg:",notnull"`
	R                    string `pg:",notnull"`
	S                    string `pg:",notnull"`
}

func (f *FEVMTransaction) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_transactions"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMTransactionList []*FEVMTransaction

func (f FEVMTransactionList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_transactions"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
