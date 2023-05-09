package messages

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Receipt struct {
	Height    int64  `pg:",pk,notnull,use_zero"` // note this is the height of the receipt not the message
	Message   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	Idx      int   `pg:",use_zero"`
	ExitCode int64 `pg:",use_zero"`
	GasUsed  int64 `pg:",use_zero"`
}

func (r *Receipt) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "receipts"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, r)
}

type Receipts []*Receipt

func (rs Receipts) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(rs) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "Receipts.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(rs)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "receipts"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(rs))
	return s.PersistModel(ctx, rs)
}
