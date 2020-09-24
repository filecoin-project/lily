package messages

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

type MessageTaskResult struct {
	Messages      Messages
	BlockMessages BlockMessages
	Receipts      Receipts
}

func (mtr *MessageTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MessageTaskResult.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskNS, tasks.MessagePoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(-1))

	start := time.Now()
	defer func() {
		stats.Record(ctx, metrics.PersistDuration.M(metrics.SinceInMilliseconds(start)))
	}()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := mtr.Messages.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := mtr.BlockMessages.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := mtr.Receipts.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		return nil
	})
}
