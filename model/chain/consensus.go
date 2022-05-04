package chain

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ChainConsensus struct {
	// explicit table name to ensure model changes don't affect task names
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"chain_consensus"`

	Height          int64  `pg:",pk,notnull,use_zero"`
	ParentStateRoot string `pg:",pk,notnull"`
	ParentTipSet    string `pg:",pk,notnull"`
	TipSet          string
}

func (c ChainConsensus) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainConsensus.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_consensus"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, c)
}

type ChainConsensusList []*ChainConsensus

func (c ChainConsensusList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainConsensusList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(c)))
	}
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
