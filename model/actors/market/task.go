package market

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MarketTaskResult struct {
	Proposals MarketDealProposals
	States    MarketDealStates
}

func (mtr *MarketTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if err := mtr.Proposals.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := mtr.States.Persist(ctx, s, version); err != nil {
		return err
	}
	return nil
}
