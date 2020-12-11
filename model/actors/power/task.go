package power

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type PowerTaskResult struct {
	ChainPowerModel *ChainPower
	ClaimStateModel PowerActorClaimList
}

func (p *PowerTaskResult) Persist(ctx context.Context, s model.StorageBatch) error {
	if p.ChainPowerModel != nil {
		if err := p.ChainPowerModel.Persist(ctx, s); err != nil {
			return err
		}
	}
	if p.ClaimStateModel != nil {
		if err := p.ClaimStateModel.Persist(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
