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

type ParsedMessage struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Cid    string `pg:",pk,notnull"`
	From   string `pg:",notnull"`
	To     string `pg:",notnull"`
	Value  string `pg:",notnull"`
	Method string `pg:",notnull"`

	Params string `pg:",type:jsonb,notnull"`
}

func (pm *ParsedMessage) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "parsed_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, pm)
}

type ParsedMessages []*ParsedMessage

func (pms ParsedMessages) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	if len(pms) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ParsedMessages.Persist", trace.WithAttributes(label.Int("count", len(pms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "parsed_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, pms)
}
