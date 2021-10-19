package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
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

func (m *MinerCurrentDeadlineInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerCurrentDeadlineInfo.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_current_deadline_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerCurrentDeadlineInfoList []*MinerCurrentDeadlineInfo

func (ml MinerCurrentDeadlineInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerCurrentDeadlineInfoList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_current_deadline_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
