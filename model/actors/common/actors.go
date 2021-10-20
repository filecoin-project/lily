package common

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type Actor struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	ID        string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`
	Code      string `pg:",notnull"`
	Head      string `pg:",notnull"`
	Balance   string `pg:",notnull"`
	Nonce     uint64 `pg:",use_zero"`
}

func (a *Actor) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if a == nil {
		// Nothing to do
		return nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "Actor.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actors"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, a)
}

// ActorList is a slice of Actors persistable in a single batch.
type ActorList []*Actor

func (actors ActorList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ActorList.Persist", trace.WithAttributes(attribute.Int("count", len(actors))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actors"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(actors))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(actors) == 0 {
		return nil
	}
	return s.PersistModel(ctx, actors)
}

type ActorState struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Head   string `pg:",pk,notnull"`
	Code   string `pg:",pk,notnull"`
	State  string `pg:",type:jsonb,notnull"`
}

func (as *ActorState) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if as == nil {
		// Nothing to do
		return nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "ActorState.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_states"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, as)
}

// ActorStateList is a list of ActorStates persistable in a single batch.
type ActorStateList []*ActorState

func (states ActorStateList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ActorStateList.Persist", trace.WithAttributes(attribute.Int("count", len(states))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_states"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(states) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(states))
	return s.PersistModel(ctx, states)
}
