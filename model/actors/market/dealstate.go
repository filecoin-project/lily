package market

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MarketDealState struct {
	Height           int64  `pg:",pk,notnull,use_zero"`
	DealID           uint64 `pg:",pk,use_zero"`
	SectorStartEpoch int64  `pg:",pk,use_zero"`
	LastUpdateEpoch  int64  `pg:",pk,use_zero"`
	SlashEpoch       int64  `pg:",pk,use_zero"`

	StateRoot string `pg:",notnull"`
}

func (ds *MarketDealState) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_states"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ds)
}

type MarketDealStates []*MarketDealState

func (dss MarketDealStates) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MarketDealStates.PersistWithTx", trace.WithAttributes(attribute.Int("count", len(dss))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_states"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(dss))
	return s.PersistModel(ctx, dss)
}
