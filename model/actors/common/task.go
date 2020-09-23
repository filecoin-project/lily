package common

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

type ActorTaskResult struct {
	Actor *Actor
	State *ActorState
}

func (a *ActorTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "ActorTaskResult.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskNS, tasks.CommonPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(-1))

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
