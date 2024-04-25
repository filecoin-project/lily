package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MinerSectorDealV2 struct {
	tableName struct{} `pg:"miner_sector_deal_v2"` // nolint: structcheck

	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",pk,use_zero"`
}

func (ds *MinerSectorDealV2) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals_v2"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ds)
}

type MinerSectorDealListV2 []*MinerSectorDealV2

func (ml MinerSectorDealListV2) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerSectorDealListV2.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals_v2"))

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
