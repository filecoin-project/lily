package market

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
)

type MarketTaskResult struct {
	Proposals MarketDealProposals
	States    MarketDealStates
}

func (mtr *MarketTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketTaskResult.Persist")
	defer span.End()
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := mtr.Proposals.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := mtr.States.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		return nil
	})
}
