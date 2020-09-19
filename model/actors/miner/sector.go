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

func NewMinerSectorInfos(res *MinerTaskResult) MinerSectorInfos {
	out := make(MinerSectorInfos, len(res.SectorChanges.Added))
	for i, added := range res.SectorChanges.Added {
		si := &MinerSectorInfo{
			MinerID:               res.Addr.String(),
			SectorID:              uint64(added.SectorNumber),
			StateRoot:             res.StateRoot.String(),
			SealedCID:             added.SealedCID.String(),
			ActivationEpoch:       int64(added.Activation),
			ExpirationEpoch:       int64(added.Expiration),
			DealWeight:            added.DealWeight.String(),
			VerifiedDealWeight:    added.VerifiedDealWeight.String(),
			InitialPledge:         added.InitialPledge.String(),
			ExpectedDayReward:     added.ExpectedDayReward.String(),
			ExpectedStoragePledge: added.ExpectedStoragePledge.String(),
		}
		out[i] = si
	}
	return out
}

type MinerSectorInfos []*MinerSectorInfo

func (msis MinerSectorInfos) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return msis.PersistWithTx(ctx, tx)
	})
}

func (msis MinerSectorInfos) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorInfos.PersistWithTx", trace.WithAttributes(label.Int("count", len(msis))))
	defer span.End()
	for _, msi := range msis {
		if err := msi.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
