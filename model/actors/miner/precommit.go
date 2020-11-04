package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type MinerPreCommitInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
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

type MinerPreCommitInfoList []*MinerPreCommitInfo

func (ml MinerPreCommitInfoList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ml.PersistWithTx(ctx, tx)
	})
}

func (ml MinerPreCommitInfoList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerPreCommitInfoList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner pre commit info list: %w")
	}
	return nil
}
