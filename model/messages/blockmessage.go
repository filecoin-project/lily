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

type BlockMessage struct {
	Block   string `pg:",pk,notnull"`
	Message string `pg:",pk,notnull"`
}

func (bm *BlockMessage) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bm).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting block message: %w", err)
	}
	return nil
}

type BlockMessages []*BlockMessage

func (bms BlockMessages) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(bms) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "BlockMessages.PersistWithTx", trace.WithAttributes(label.Int("count", len(bms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message/blockmessage"))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	if _, err := tx.ModelContext(ctx, &bms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting block messages: %w", err)
	}
	return nil
}
