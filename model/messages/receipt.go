package messages

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type Receipt struct {
	Height    int64  `pg:",pk,notnull,use_zero"` // note this is the height of the receipt not the message
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
	if len(rs) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "Receipts.PersistWithTx", trace.WithAttributes(label.Int("count", len(rs))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message/receipt"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if _, err := tx.ModelContext(ctx, &rs).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting receipts: %w", err)
	}
	return nil
}
