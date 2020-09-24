package reward

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/tasks"
)

type ChainReward struct {
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

func (r *ChainReward) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainReward.PersistWithTx")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskNS, tasks.RewardPoolName))
	stats.Record(ctx, metrics.TaskQueueLen.M(-1))

	start := time.Now()
	defer stats.Record(ctx, metrics.PersistDuration.M(metrics.SinceInMilliseconds(start)))

	if _, err := tx.ModelContext(ctx, r).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

func (r *ChainReward) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return r.PersistWithTx(ctx, tx)
	})
}
