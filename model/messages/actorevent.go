package messages

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ActorEvent struct {
	tableName struct{} `pg:"actor_events"`

	Height     int64  `pg:",pk,notnull,use_zero"`
	StateRoot  string `pg:",pk,notnull"`
	MessageCid string `pg:",pk,notnull"`
	EventIndex int64  `pg:",pk,notnull,use_zero"`

	Emitter string `pg:",notnull"`
	Flags   []byte `pg:",notnull"`
	Key     string `pg:",notnull"`
	Value   []byte `pg:",notnull"`
}

func (a *ActorEvent) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_events"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, a)
}

type ActorEventList []*ActorEvent

func (al ActorEventList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(al) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_events"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(al))
	return s.PersistModel(ctx, al)
}
