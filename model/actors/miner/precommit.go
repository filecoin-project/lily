package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/xerrors"
)

type MinerPreCommitInfo struct {
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	SealedCID       string `pg:",notnull"`
	SealRandEpoch   int64  `pg:",use_zero"`
	ExpirationEpoch int64  `pg:",use_zero"`

	PreCommitDeposit   string `pg:",notnull"`
	PreCommitEpoch     int64  `pg:",use_zero"`
	DealWeight         string `pg:",notnull"`
	VerifiedDealWeight string `pg:",notnull"`

	IsReplaceCapacity      bool
	ReplaceSectorDeadline  uint64 `pg:",use_zero"`
	ReplaceSectorPartition uint64 `pg:",use_zero"`
	ReplaceSectorNumber    uint64 `pg:",use_zero"`
}

func (mpi *MinerPreCommitInfo) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, mpi).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner precommit info: %w", err)
	}
	return nil
}

func NewMinerPreCommitInfos(res *MinerTaskResult) MinerPreCommitInfos {
	out := make(MinerPreCommitInfos, len(res.PreCommitChanges.Added))
	for i, added := range res.PreCommitChanges.Added {
		pc := &MinerPreCommitInfo{
			MinerID:   res.Addr.String(),
			SectorID:  uint64(added.Info.SectorNumber),
			StateRoot: res.StateRoot.String(),

			SealedCID:       added.Info.SealedCID.String(),
			SealRandEpoch:   int64(added.Info.SealRandEpoch),
			ExpirationEpoch: int64(added.Info.Expiration),

			PreCommitDeposit:   added.PreCommitDeposit.String(),
			PreCommitEpoch:     int64(added.PreCommitEpoch),
			DealWeight:         added.DealWeight.String(),
			VerifiedDealWeight: added.VerifiedDealWeight.String(),

			IsReplaceCapacity:      added.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  added.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: added.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(added.Info.ReplaceSectorNumber),
		}
		out[i] = pc
	}
	return out
}

type MinerPreCommitInfos []*MinerPreCommitInfo

func (mpis MinerPreCommitInfos) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return mpis.PersistWithTx(ctx, tx)
	})
}

func (mpis MinerPreCommitInfos) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MinerPreCommitInfos.PersistWithTx", opentracing.Tags{"count": len(mpis)})
	defer span.Finish()
	for _, mpi := range mpis {
		if err := mpi.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
