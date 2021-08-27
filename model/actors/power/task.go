package power

import (
	"context"

	"github.com/filecoin-project/lily/model"
)

type PowerTaskResult struct {
	ChainPowerModel *ChainPower
	ClaimStateModel PowerActorClaimList
}

func (p *PowerTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if p.ChainPowerModel != nil {
		if err := p.ChainPowerModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if p.ClaimStateModel != nil {
		if err := p.ClaimStateModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}
