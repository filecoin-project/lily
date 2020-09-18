package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
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
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ms.PersistWithTx(ctx, tx)
	})

}

func (ms *MinerState) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MinerTaskResult.PersistWithTx")
	defer span.Finish()
	if _, err := tx.ModelContext(ctx, ms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return nil
}
