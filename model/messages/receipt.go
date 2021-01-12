package messages

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type Receipt struct {
	Height    int64  `pg:",pk,notnull,use_zero"` // note this is the height of the receipt not the message
	Message   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	Idx      int   `pg:",use_zero"`
	ExitCode int64 `pg:",use_zero"`
	GasUsed  int64 `pg:",use_zero"`
}

func (r *Receipt) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, r)
}

type Receipts []*Receipt

func (rs Receipts) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(rs) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "Receipts.Persist", trace.WithAttributes(label.Int("count", len(rs))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message/receipt"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, rs)
}
