package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

type Receipt struct {
	Message   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	Idx      int   `pg:",use_zero"`
	ExitCode int64 `pg:",use_zero"`
	GasUsed  int64 `pg:",use_zero"`

	Return []byte
}

func (r *Receipt) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, r).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting receipt: %w", err)
	}
	return nil
}

type Receipts []*Receipt

func (rs Receipts) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "Receipts.PersistWithTx", trace.WithAttributes(label.Int("count", len(rs))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskNS, fmt.Sprintf("%s_%s", tasks.MessagePoolName, "receipts")))
	start := time.Now()
	defer func() {
		stats.Record(ctx, metrics.PersistDuration.M(metrics.SinceInMilliseconds(start)))
	}()

	for _, r := range rs {
		if err := r.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
