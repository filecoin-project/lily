package reward

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ChainRewardDistribution struct {
	tableName struct{} `pg:"chain_reward_distributions"` // nolint: structcheck

	Height       int64  `pg:",pk,notnull,use_zero"`
	TipSetKey    string `pg:"tipset_key,pk,notnull"`
	Distribution string `pg:",type:jsonb,notnull"`
}

func (r *ChainRewardDistribution) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "ChainRewardDistribution.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "chain_reward_distributions"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, r)
}
