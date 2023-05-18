package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMBlockHeader struct {
	tableName struct{} `pg:"fevm_block_header"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`

	// ETH Hash
	Hash string `pg:",notnull"`

	// Parent Block ETH Hash
	ParentHash string `pg:",notnull"`

	Miner string `pg:",notnull"`

	StateRoot string `pg:",notnull"`

	TransactionsRoot string `pg:",notnull"`
	ReceiptsRoot     string `pg:",notnull"`
	Difficulty       uint64 `pg:",use_zero"`
	Number           uint64 `pg:",use_zero"`
	GasLimit         uint64 `pg:",use_zero"`
	GasUsed          uint64 `pg:",use_zero"`
	Timestamp        uint64 `pg:",use_zero"`
	ExtraData        string `pg:",notnull"`
	MixHash          string `pg:",notnull"`
	Nonce            string `pg:",notnull"`
	BaseFeePerGas    string `pg:",notnull"`
	Size             uint64 `pg:",use_zero"`
	Sha3Uncles       string `pg:",notnull"`
}

func (f *FEVMBlockHeader) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_block_header"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMBlockHeaderList []*FEVMBlockHeader

func (f FEVMBlockHeaderList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_block_header"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
