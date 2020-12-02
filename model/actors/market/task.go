package market

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type MarketTaskResult struct {
	Proposals MarketDealProposals
	States    MarketDealStates
}

func (mtr *MarketTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketTaskResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return mtr.PersistWithTx(ctx, tx)
	})
}

func (mtr *MarketTaskResult) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if err := mtr.Proposals.PersistWithTx(ctx, tx); err != nil {
		return fmt.Errorf("persisting market deal proposal: %w", err)
	}
	if err := mtr.States.PersistWithTx(ctx, tx); err != nil {
		return fmt.Errorf("persisting market deal state: %w", err)
	}
	return nil
}
