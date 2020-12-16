package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerLockedFund struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	LockedFunds       string `pg:"type:numeric,notnull"`
	InitialPledge     string `pg:"type:numeric,notnull"`
	PreCommitDeposits string `pg:"type:numeric,notnull"`
}

func (m *MinerLockedFund) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerLockedFund.Persist")
	defer span.End()
	return s.PersistModel(ctx, m)
}

type MinerLockedFundsList []*MinerLockedFund

func (ml MinerLockedFundsList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerLockedFundsList.Persist")
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}
