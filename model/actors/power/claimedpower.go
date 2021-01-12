package power

import (
	"context"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/model"
)

type PowerActorClaim struct {
	Height          int64  `pg:",pk,notnull,use_zero"`
	MinerID         string `pg:",pk,notnull"`
	StateRoot       string `pg:",pk,notnull"`
	RawBytePower    string `pg:"type:numeric,notnull"`
	QualityAdjPower string `pg:"type:numeric,notnull"`
}

func (p *PowerActorClaim) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaim.Persist")
	defer span.End()
	return s.PersistModel(ctx, p)
}

type PowerActorClaimList []*PowerActorClaim

func (pl PowerActorClaimList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerActorClaimList.Persist")
	defer span.End()
	if len(pl) == 0 {
		return nil
	}
	return s.PersistModel(ctx, pl)
}
