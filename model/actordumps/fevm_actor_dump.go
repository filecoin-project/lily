package actordumps

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMActorDump struct {
	tableName struct{} `pg:"fevm_actor_dumps"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Actor address.
	ActorID string `pg:",pk,notnull"`
	// Actor Address in ETH.
	EthAddress string `pg:",notnull"`
	// Contract Bytecode.
	ByteCode string `pg:",notnull"`
	// Contract Bytecode is encoded in hash by Keccak256.
	ByteCodeHash string `pg:",notnull"`
	// Balance of EVM actor in attoFIL.
	Balance string `pg:"type:numeric,notnull"`
	// The next actor nonce that is expected to appear on chain.
	Nonce uint64 `pg:",pk,use_zero"`
	// Actor Name
	ActorName string `pg:",notnull"`
}

func (f *FEVMActorDump) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_dumps"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMActorDumpList []*FEVMActorDump

func (f FEVMActorDumpList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_dumps"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
