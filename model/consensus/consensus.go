package consensus

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type ChainConsensus struct {
	Height       int64  `pg:",pk,notnull,use_zero"`
	StateRoot    string `pg:",pk,notnull"`
	ParentTipSet string `pg:",pk,notnull"`
	TipSet       string
}

func (c ChainConsensus) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainConsensus.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_consensus"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, c)
}

type ChainConsensusList []*ChainConsensus

func (c ChainConsensusList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainConsensusList.Persist", trace.WithAttributes(label.Int("count", len(c))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_consensus"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(c))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(c) == 0 {
		return nil
	}
	return s.PersistModel(ctx, c)
}
