package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMBlockHeader struct {
	tableName struct{} `pg:"fevm_block_headers"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// ETH Hash
	Hash string `pg:",notnull"`
	// Parent Block ETH Hash
	ParentHash string `pg:",notnull"`
	// ETH Address of the miner who mined this block.
	Miner string `pg:",notnull"`
	// Block state root ETH hash.
	StateRoot string `pg:",notnull"`
	// Set to a hardcoded value which is used by some clients to determine if has no transactions.
	TransactionsRoot string `pg:",notnull"`
	// Hash of the transaction receipts trie.
	ReceiptsRoot string `pg:",notnull"`
	// ETH mining difficulty.
	Difficulty uint64 `pg:",use_zero"`
	// The number of the current block.
	Number uint64 `pg:",use_zero"`
	// Maximum gas allowed in this block.
	GasLimit uint64 `pg:",use_zero"`
	// The actual amount of gas used in this block.
	GasUsed uint64 `pg:",use_zero"`
	// The block time.
	Timestamp uint64 `pg:",use_zero"`
	// Arbitrary additional data as raw bytes.
	ExtraData string `pg:",notnull"`
	MixHash   string `pg:",notnull"`
	Nonce     string `pg:",notnull"`
	// The base fee value.
	BaseFeePerGas string `pg:",notnull"`
	// Block size.
	Size       uint64 `pg:",use_zero"`
	Sha3Uncles string `pg:",notnull"`
}

func (f *FEVMBlockHeader) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_block_headers"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMBlockHeaderList []*FEVMBlockHeader

func (f FEVMBlockHeaderList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_block_headers"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
