package common

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// Actor on chain that were added or updated at an epoch.
// Associates the actor's state root CID (head) with the chain state root CID from which it decends.
// Includes account ID nonce and balance at each state.
type Actor struct {
	// Epoch when this actor was created or updated.
	Height int64 `pg:",pk,notnull,use_zero"`
	// ID Actor address.
	ID string `pg:",pk,notnull"`
	// CID of the state root when this actor was created or changed.
	StateRoot string `pg:",pk,notnull"`
	// Human-readable identifier for the type of the actor.
	Code string `pg:",notnull"`
	// CID identifier for the type of the actor.
	CodeCID string `pg:",notnull"`
	// CID of the root of the state tree for the actor.
	Head string `pg:",notnull"`
	// Balance of Actor in attoFIL.
	Balance string `pg:",notnull"`
	// The next Actor nonce that is expected to appear on chain.
	Nonce uint64 `pg:",use_zero"`
	// Top level of state data as json.
	State string `pg:",type:jsonb"`
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
	ctx, span := otel.Tracer("").Start(ctx, "ActorList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(actors)))
	}
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

// ActorState that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor’s state.
type ActorState struct {
	// Epoch when this actor was created or updated.
	Height int64 `pg:",pk,notnull,use_zero"`
	// CID of the root of the state tree for the actor.
	Head string `pg:",pk,notnull"`
	// CID identifier for the type of the actor.
	Code string `pg:",pk,notnull"`
	// Top level of state data as json.
	State string `pg:",type:jsonb,notnull"`
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
	ctx, span := otel.Tracer("").Start(ctx, "ActorStateList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(states)))
	}
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

type ActorCode struct {
	// CID of the actor from builtin actors.
	CID string `pg:",pk,notnull"`
	// Human-readable identifier for the actor.
	Code string `pg:",pk,notnull"`
}

type ActorCodeList []*ActorCode

func (a *ActorCode) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if a == nil {
		// Nothing to do
		return nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "ActorCode.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_codes"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, a)
}

func (acl ActorCodeList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_codes"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(acl))
	return s.PersistModel(ctx, acl)
}

type ActorMethod struct {
	Family     string `pg:",pk,notnull"`
	MethodName string `pg:",notnull"`
	Method     uint64 `pg:",pk,notnull"`
}

type ActorMethodList []*ActorMethod

func (a *ActorMethod) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if a == nil {
		// Nothing to do
		return nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "ActorMethod.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_methods"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, a)
}

func (acl ActorMethodList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "actor_methods"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(acl))
	return s.PersistModel(ctx, acl)
}
