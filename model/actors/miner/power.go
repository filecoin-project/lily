package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

func NewMinerPowerModel(res *MinerTaskResult) *MinerPower {
	return &MinerPower{
		MinerID:              res.Addr.String(),
		StateRoot:            res.StateRoot.String(),
		RawBytePower:         res.Power.MinerPower.RawBytePower.String(),
		QualityAdjustedPower: res.Power.MinerPower.QualityAdjPower.String(),
	}
}

type MinerPower struct {
	MinerID              string `pg:",pk,notnull"`
	StateRoot            string `pg:",pk,notnull"`
	RawBytePower         string `pg:",notnull"`
	QualityAdjustedPower string `pg:",notnull"`
}

func (mp *MinerPower) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ModelContext(ctx, mp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return tx.CommitContext(ctx)
}

func (mp *MinerPower) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, mp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return nil
}
