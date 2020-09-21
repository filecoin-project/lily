package common

import (
	"context"
	"github.com/go-pg/pg/v10"
)

type ActorTaskResult struct {
	Actor *Actor
	State *ActorState
}

func (a *ActorTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := a.Actor.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := a.State.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		return nil
	})
}
