package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMActorStats struct {
	tableName struct{} `pg:"fevm_actor_stats"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`

	// Balance of EVM actor in attoFIL.
	ContractBalance string `pg:",notnull"`
	// Balance of Eth account actor in attoFIL.
	EthAccountBalance string `pg:",notnull"`
	// Balance of Placeholder Actor in attoFIL.
	PlaceholderBalance string `pg:",notnull"`

	// number of contracts
	ContractCount uint64 `pg:",use_zero"`
	// number of unique contracts
	UniqueContractCount uint64 `pg:",use_zero"`
	// number of Eth account actors
	EthAccountCount uint64 `pg:",use_zero"`
	// number of placeholder actors
	PlaceholderCount uint64 `pg:",use_zero"`
}

func (f *FEVMActorStats) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_stats"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMActorStatsList []*FEVMActorStats

func (f FEVMActorStatsList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_stats"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
