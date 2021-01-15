package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerSectorInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	SealedCID string `pg:",notnull"`

	ActivationEpoch int64 `pg:",use_zero"`
	ExpirationEpoch int64 `pg:",use_zero"`

	DealWeight         string `pg:"type:numeric,notnull"`
	VerifiedDealWeight string `pg:"type:numeric,notnull"`

	InitialPledge         string `pg:"type:numeric,notnull"`
	ExpectedDayReward     string `pg:"type:numeric,notnull"`
	ExpectedStoragePledge string `pg:"type:numeric,notnull"`
}

func (msi *MinerSectorInfo) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, msi)
}

type MinerSectorInfoList []*MinerSectorInfo

func (ml MinerSectorInfoList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorInfoList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}
