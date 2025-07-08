package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MinerCronFee struct {
	tableName struct{} `pg:"miner_cron_fees"` // nolint: structcheck

	Height  int64  `pg:",pk,notnull,use_zero"`
	Address string `pg:",pk,notnull"`

	Burn    string `pg:"type:numeric,notnull"`
	Fee     string `pg:"type:numeric,notnull"`
	Penalty string `pg:"type:numeric,notnull"`
}

func (m *MinerCronFee) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerCronFee.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_cron_fees"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerCronFeeList []*MinerCronFee

func (ml MinerCronFeeList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerCronFeeList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_cron_fees"))

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
