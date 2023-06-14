package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMContract struct {
	tableName struct{} `pg:"fevm_contracts"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Actor address.
	ActorID string `pg:",notnull"`
	// Actor Address in ETH
	EthAddress string `pg:",notnull"`
	// Contract Bytecode.
	ByteCode string `pg:",notnull"`
	// Contract Bytecode is encoded in hash by Keccak256.
	ByteCodeHash string `pg:",notnull"`
	// Balance of EVM actor in attoFIL.
	Balance string `pg:"type:numeric,notnull"`
	// The next actor nonce that is expected to appear on chain.
	Nonce uint64 `pg:",use_zero"`
}

func (f *FEVMContract) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_contracts"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMContractList []*FEVMContract

func (f FEVMContractList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_contracts"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
