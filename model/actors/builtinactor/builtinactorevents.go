package builtinactor

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type BuiltInActorEvent struct {
	tableName struct{} `pg:"builtin_actor_events"` // nolint: structcheck

	Height       int64  `pg:",pk,notnull,use_zero"`
	Cid          string `pg:",pk,notnull"`
	Emitter      string `pg:",pk,notnull"`
	EventType    string `pg:",pk,notnull"`
	EventEntries string `pg:",type:jsonb"`
	EventPayload string `pg:",type:jsonb"`
	EventIdx     int64  `pg:",pk,notnull"`
}

func (ds *BuiltInActorEvent) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "builtin_actor_events"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ds)
}

type BuiltInActorEvents []*BuiltInActorEvent

func (dss BuiltInActorEvents) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "BuiltInActorEvents.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dss)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "builtin_actor_events"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(dss))
	return s.PersistModel(ctx, dss)
}
