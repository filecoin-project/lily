package market

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MarketTaskResult struct {
	Proposals MarketDealProposals
	States    MarketDealStates
}

func (mtr *MarketTaskResult) Persist(ctx context.Context, s model.StorageBatch) error {
	if err := mtr.Proposals.Persist(ctx, s); err != nil {
		return err
	}
	if err := mtr.States.Persist(ctx, s); err != nil {
		return err
	}
	return nil
}
