package common

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type ActorTaskResult struct {
	Actor *Actor
	State *ActorState
}

func (a *ActorTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorTaskResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

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

func (a *ActorTaskResult) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if err := a.Actor.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if err := a.State.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	return nil
}
