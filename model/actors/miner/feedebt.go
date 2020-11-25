package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

type MinerFeeDebt struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	FeeDebt string `pg:",notnull"`
}

func (m *MinerFeeDebt) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerFeeDebt.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, m).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner fee debt: %w", err)
	}
	return nil
}

type MinerFeeDebtList []*MinerFeeDebt

func (ml MinerFeeDebtList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerFeeDebtList.PersistWithTx")
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner fee debt list: %w", err)
	}
	return nil
}
