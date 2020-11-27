package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

type MinerCurrentDeadlineInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	DeadlineIndex uint64 `pg:",notnull,use_zero"`
	PeriodStart   int64  `pg:",notnull,use_zero"`
	Open          int64  `pg:",notnull,use_zero"`
	Close         int64  `pg:",notnull,use_zero"`
	Challenge     int64  `pg:",notnull,use_zero"`
	FaultCutoff   int64  `pg:",notnull,use_zero"`
}

func (m *MinerCurrentDeadlineInfo) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerCurrentDeadlineInfo.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, m).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner current deadline: %w", err)
	}
	return nil
}

type MinerCurrentDeadlineInfoList []*MinerCurrentDeadlineInfo

func (ml MinerCurrentDeadlineInfoList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerCurrentDeadlineInfoList.PersistWithTx")
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner current deadline list: %w", err)
	}
	return nil
}
