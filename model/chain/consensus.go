package chain

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ChainConsensus struct {
	Height          int64  `pg:",pk,notnull,use_zero"`
	ParentStateRoot string `pg:",pk,notnull"`
	ParentTipSet    string `pg:",pk,notnull"`
	TipSet          string
}

func (c ChainConsensus) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainConsensus.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_consensus"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)

	return s.PersistModel(ctx, c)
}

type ChainConsensusList []*ChainConsensus

func (c ChainConsensusList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainConsensusList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(c)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_consensus"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(c))

	if len(c) == 0 {
		return nil
	}
	return s.PersistModel(ctx, c)
}
