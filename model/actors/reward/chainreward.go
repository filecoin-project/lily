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
	CumSumBaseline                    string `pg:",notnull"`
	CumSumRealized                    string `pg:",notnull"`
	EffectiveBaselinePower            string `pg:",notnull"`
	NewBaselinePower                  string `pg:",notnull"`
	NewRewardSmoothedPositionEstimate string `pg:",notnull"`
	NewRewardSmoothedVelocityEstimate string `pg:",notnull"`
	TotalMinedReward                  string `pg:",notnull"`

	NewReward            string `pg:",use_zero"`
	EffectiveNetworkTime int64  `pg:",use_zero"`
}

func (r *ChainReward) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainReward.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, r)
}
