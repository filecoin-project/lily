package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerSectorDeal struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",pk,use_zero"`
}

func (ds *MinerSectorDeal) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, ds)
}

type MinerSectorDealList []*MinerSectorDeal

func (ml MinerSectorDealList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorDealList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_deals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
