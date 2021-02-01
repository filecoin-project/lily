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

type Message struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Cid    string `pg:",pk,notnull"`

	From       string `pg:",notnull"`
	To         string `pg:",notnull"`
	Value      string `pg:",notnull"`
	GasFeeCap  string `pg:",notnull"`
	GasPremium string `pg:",notnull"`

	GasLimit  int64  `pg:",use_zero"`
	SizeBytes int    `pg:",use_zero"`
	Nonce     uint64 `pg:",use_zero"`
	Method    uint64 `pg:",use_zero"`
}

func (m *Message) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, m)
}

type Messages []*Message

func (ms Messages) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(ms) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "Messages.Persist", trace.WithAttributes(label.Int("count", len(ms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, ms)
}
