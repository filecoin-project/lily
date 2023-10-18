package market

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type MarketDealState struct {
	Height           int64  `pg:",pk,notnull,use_zero"`
	DealID           uint64 `pg:",pk,use_zero"`
	SectorStartEpoch int64  `pg:",use_zero"`
	LastUpdateEpoch  int64  `pg:",use_zero"`
	SlashEpoch       int64  `pg:",use_zero"`

	StateRoot string `pg:",pk,notnull"`
}

func (ds *MarketDealState) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_states"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ds)
}

type MarketDealStates []*MarketDealState

func (dss MarketDealStates) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MarketDealStates.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dss)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "market_deal_states"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(dss))
	return s.PersistModel(ctx, dss)
}
