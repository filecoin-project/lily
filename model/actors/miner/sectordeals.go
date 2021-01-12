package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerSectorDeal struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",pk,use_zero"`
}

func (ds *MinerSectorDeal) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, ds)
}

type MinerSectorDealList []*MinerSectorDeal

func (ml MinerSectorDealList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorDealList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}
