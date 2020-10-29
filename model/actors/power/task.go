package power

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type PowerTaskResult struct {
	ChainPowerModel *ChainPower
	ClaimStateModel PowerActorClaimList
}

func (p *PowerTaskResult) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if p.ChainPowerModel != nil {
		if err := p.ChainPowerModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if p.ClaimStateModel != nil {
		if err := p.ClaimStateModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func (p *PowerTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "PowerTaskResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return p.PersistWithTx(ctx, tx)
	})
}
