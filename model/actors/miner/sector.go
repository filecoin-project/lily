package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type MinerSectorInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	SealedCID string `pg:",notnull"`

	ActivationEpoch int64 `pg:",use_zero"`
	ExpirationEpoch int64 `pg:",use_zero"`

	DealWeight         string `pg:",notnull"`
	VerifiedDealWeight string `pg:",notnull"`

	InitialPledge         string `pg:",notnull"`
	ExpectedDayReward     string `pg:",notnull"`
	ExpectedStoragePledge string `pg:",notnull"`
}

func (msi *MinerSectorInfo) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, msi).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner precommit info: %w", err)
	}
	return nil
}

type MinerSectorInfoList []*MinerSectorInfo

func (ml MinerSectorInfoList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ml.PersistWithTx(ctx, tx)
	})
}

func (ml MinerSectorInfoList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorInfoList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner sector info list: %w", err)
	}
	return nil
}
