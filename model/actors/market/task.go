package market

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

type MarketTaskResult struct {
	Proposals MarketDealProposals
	States    MarketDealStates
}

func (mtr *MarketTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketTaskResult.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskNS, tasks.MarketPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(-1))

	start := time.Now()
	defer func() {
		stats.Record(ctx, metrics.PersistDuration.M(metrics.SinceInMilliseconds(start)))
	}()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := mtr.Proposals.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persisting market deal proposal: %w", err)
		}
		if err := mtr.States.PersistWithTx(ctx, tx); err != nil {
			return fmt.Errorf("persisting market deal state: %w", err)
		}
		return nil
	})
}
