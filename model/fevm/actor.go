package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMActor struct {
	tableName struct{} `pg:"fevm_actors"` // nolint: structcheck
	// Epoch when this actor was deployed.
	Height int64 `pg:",pk,notnull,use_zero"`
	// ID Actor address.
	ID string `pg:",pk,notnull"`
	// ETH Address
	EthAddress string `pg:",pk,notnull"`
	// CID of the state root when this actor was created or changed.
	StateRoot string `pg:",pk,notnull"`
	// Human-readable identifier for the type of the actor.
	Code string `pg:",notnull"`
	// CID of the root of the state tree for the actor.
	Head string `pg:",notnull"`
	// Top level of state data as json.
	State string `pg:",type:jsonb"`
	// CID identifier for the type of the actor.
	CodeCID string `pg:",notnull"`
}

func (f *FEVMActor) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actors"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMActorList []*FEVMActor

func (f FEVMActorList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_actors"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
