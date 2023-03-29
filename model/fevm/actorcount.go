package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMActorCount struct {
	tableName struct{} `pg:"fevm_actor_count"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// Balance of EVM Actor in attoFIL.
	EVMBalance string `pg:",notnull"`
	// Balance of Eth Account Actor in attoFIL.
	EthAccountBalance string `pg:",notnull"`
	// Balance of Placeholder Actor in attoFIL.
	PlaceholderBalance string `pg:",notnull"`
}

func (f *FEVMActorCount) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_count"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMActorCountList []*FEVMActorCount

func (f FEVMActorCountList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actor_count"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
