package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerFeeDebt struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	FeeDebt string `pg:"type:numeric,notnull"`
}

func (m *MinerFeeDebt) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerFeeDebt.Persist")
	defer span.End()
	return s.PersistModel(ctx, m)
}

type MinerFeeDebtList []*MinerFeeDebt

func (ml MinerFeeDebtList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerFeeDebtList.Persist")
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}
