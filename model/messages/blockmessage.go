package messages

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type BlockMessage struct {
	Height  int64  `pg:",pk,notnull,use_zero"`
	Block   string `pg:",pk,notnull"`
	Message string `pg:",pk,notnull"`
}

func (bm *BlockMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "block_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, bm)
}

type BlockMessages []*BlockMessage

func (bms BlockMessages) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(bms) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "BlockMessages.Persist", trace.WithAttributes(attribute.Int("count", len(bms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "block_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(bms))
	return s.PersistModel(ctx, bms)
}
