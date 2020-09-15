package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

func NewMinerStateModel(res *MinerTaskResult) *MinerState {
	return &MinerState{
		MinerID:    res.Addr.String(),
		OwnerID:    res.Info.Owner.String(),
		WorkerID:   res.Info.Worker.String(),
		PeerID:     res.Info.PeerId,
		SectorSize: res.Info.SectorSize.ShortString(),
	}
}

type MinerState struct {
	MinerID    string `pg:",pk,notnull"`
	OwnerID    string `pg:",notnull"`
	WorkerID   string `pg:",notnull"`
	PeerID     []byte
	SectorSize string `pg:",notnull"`
}

func (ms *MinerState) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.ModelContext(ctx, ms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return tx.CommitContext(ctx)
}

func (ms *MinerState) PersistWitTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return nil
}
