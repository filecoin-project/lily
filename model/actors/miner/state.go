package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

func NewMinerStateModel(res *MinerTaskResult) *MinerState {
	ms := &MinerState{
		Height:     int64(res.Height),
		MinerID:    res.Addr.String(),
		OwnerID:    res.Info.Owner.String(),
		WorkerID:   res.Info.Worker.String(),
		SectorSize: res.Info.SectorSize.ShortString(),
	}

	if res.Info.PeerId != nil {
		ms.PeerID = res.Info.PeerId.String()
	}

	return ms
}

type MinerState struct {
	Height     int64  `pg:",pk,notnull,use_zero"`
	MinerID    string `pg:",pk,notnull"`
	OwnerID    string `pg:",notnull"`
	WorkerID   string `pg:",notnull"`
	PeerID     string
	SectorSize string `pg:",notnull"`
}

func (ms *MinerState) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ms.PersistWithTx(ctx, tx)
	})
}

func (ms *MinerState) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskResult.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, ms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner power: %w", err)
	}
	return nil
}
