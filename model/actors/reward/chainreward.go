package reward

import (
	"context"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type ChainReward struct {
	Height                            int64  `pg:",pk,notnull,use_zero"`
	StateRoot                         string `pg:",pk,notnull"`
	CumSumBaseline                    string `pg:"type:numeric,notnull"`
	CumSumRealized                    string `pg:"type:numeric,notnull"`
	EffectiveBaselinePower            string `pg:"type:numeric,notnull"`
	NewBaselinePower                  string `pg:"type:numeric,notnull"`
	NewRewardSmoothedPositionEstimate string `pg:"type:numeric,notnull"`
	NewRewardSmoothedVelocityEstimate string `pg:"type:numeric,notnull"`
	TotalMinedReward                  string `pg:"type:numeric,notnull"`

	NewReward            string `pg:"type:numeric,use_zero"`
	EffectiveNetworkTime int64  `pg:",use_zero"`
}

func (r *ChainReward) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainReward.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, r)
}
