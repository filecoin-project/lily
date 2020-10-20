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

type ParsedMessage struct {
	Cid string `pg:",pk,notnull"`

	Height int64  `pg:",use_zero"`
	From   string `pg:",notnull"`
	To     string `pg:",notnull"`
	Value  string `pg:",notnull"`
	Method string `pg:",notnull"`

	Params string `pg:",type:jsonb,notnull"`
}

func (bm *ParsedMessage) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bm).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting block message: %w", err)
	}
	return nil
}

type ParsedMessages []*ParsedMessage

func (pms ParsedMessages) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(pms) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ParsedMessages.PersistWithTx", trace.WithAttributes(label.Int("count", len(pms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message/parsed"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if _, err := tx.ModelContext(ctx, &pms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting parsed messages: %w", err)
	}
	return nil
}
