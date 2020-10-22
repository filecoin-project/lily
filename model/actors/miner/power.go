package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

func NewMinerPowerModel(res *MinerTaskResult) *MinerPower {
	return &MinerPower{
		Height:               int64(res.Height),
		MinerID:              res.Addr.String(),
		StateRoot:            res.StateRoot.String(),
		RawBytePower:         res.Power.MinerPower.RawBytePower.String(),
		QualityAdjustedPower: res.Power.MinerPower.QualityAdjPower.String(),
	}
}

type MinerPower struct {
	Height               int64  `pg:",pk,notnull,use_zero"`
	MinerID              string `pg:",pk,notnull"`
	StateRoot            string `pg:",pk,notnull"`
	RawBytePower         string `pg:",notnull"`
	QualityAdjustedPower string `pg:",notnull"`
}

func (mp *MinerPower) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return mp.PersistWithTx(ctx, tx)
	})
}

func (mp *MinerPower) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerPower.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, mp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return nil
}
