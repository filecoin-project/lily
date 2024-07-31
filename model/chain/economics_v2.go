package chain

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ChainEconomicsV2 struct {
	tableName           struct{} `pg:"chain_economics"` // nolint: structcheck
	Height              int64    `pg:",pk,notnull,use_zero"`
	ParentStateRoot     string   `pg:",pk,notnull"`
	CirculatingFilV2    string   `pg:"type:numeric,notnull"`
	VestedFil           string   `pg:"type:numeric,notnull"`
	MinedFil            string   `pg:"type:numeric,notnull"`
	BurntFil            string   `pg:"type:numeric,notnull"`
	LockedFilV2         string   `pg:"type:numeric,notnull"`
	FilReserveDisbursed string   `pg:"type:numeric,notnull"`
}

func (c *ChainEconomicsV2) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_economics"))

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, c)
}

type ChainEconomicsV2List []*ChainEconomicsV2

func (l ChainEconomicsV2List) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "ChainEconomicsV2List.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_economics_v2"))

	metrics.RecordCount(ctx, metrics.PersistModel, len(l))
	return s.PersistModel(ctx, l)
}
