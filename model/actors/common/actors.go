package common

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
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

func (a *Actor) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "Actor.Persist")
	defer span.End()
	return s.PersistModel(ctx, a)
}

// ActorList is a slice of Actors persistable in a single batch.
type ActorList []*Actor

// Persist
func (actors ActorList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorList.Persist", trace.WithAttributes(label.Int("count", len(actors))))
	defer span.End()

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

// PersistWithTx inserts the batch using the given transaction.
func (as *ActorState) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorState.Persist")
	defer span.End()
	return s.PersistModel(ctx, as)
}

// ActorStateList is a list of ActorStates persistable in a single batch.
type ActorStateList []*ActorState

// PersistWithTx inserts the batch using the given transaction.
func (states ActorStateList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorStateList.Persist", trace.WithAttributes(label.Int("count", len(states))))
	defer span.End()

	if len(states) == 0 {
		return nil
	}
	return s.PersistModel(ctx, states)
}
