package power

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type PowerActorClaim struct {
	Height          int64  `pg:",pk,notnull,use_zero"`
	MinerID         string `pg:",pk,notnull"`
	StateRoot       string `pg:",pk,notnull"`
	RawBytePower    string `pg:",notnull"`
	QualityAdjPower string `pg:",notnull"`
}

func (p *PowerActorClaim) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaim.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "power_actor_claims"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, p)
}

type PowerActorClaimList []*PowerActorClaim

func (pl PowerActorClaimList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaimList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "power_actor_claims"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(pl) == 0 {
		return nil
	}
	return s.PersistModel(ctx, pl)
}
