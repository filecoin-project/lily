package common

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
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

func (a *Actor) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "Actor.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, a).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

// ActorList is a slice of Actors persistable in a single batch.
type ActorList []*Actor

// Persist
func (actors ActorList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorList.PersistWithTx", trace.WithAttributes(label.Int("count", len(actors))))
	defer span.End()

	if len(actors) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &actors).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting actors: %w", err)
	}
	return nil
}

type ActorState struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Head   string `pg:",pk,notnull"`
	Code   string `pg:",pk,notnull"`
	State  string `pg:",type:jsonb,notnull"`
}

// PersistWithTx inserts the batch using the given transaction.
func (s *ActorState) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorState.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, s).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

// ActorStateList is a list of ActorStates persistable in a single batch.
type ActorStateList []*ActorState

// PersistWithTx inserts the batch using the given transaction.
func (states ActorStateList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorStateList.PersistWithTx", trace.WithAttributes(label.Int("count", len(states))))
	defer span.End()

	if len(states) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &states).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting actorStates: %w", err)
	}
	return nil
}
