package genesis

import (
	"context"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/go-pg/pg/v10"
)

type ProcessGenesisSingletonResult struct {
	minerResults []*GenesisMinerTaskResult
	marketResult *GenesisMarketTaskResult
}

func (r *ProcessGenesisSingletonResult) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		for _, res := range r.minerResults {
			if err := res.StateModel.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.PowerModel.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.SectorModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := res.DealModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		if r.marketResult != nil {
			if err := r.marketResult.DealModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := r.marketResult.ProposalModesl.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ProcessGenesisSingletonResult) AddMiner(m *GenesisMinerTaskResult) {
	r.minerResults = append(r.minerResults, m)
}

func (r *ProcessGenesisSingletonResult) SetMarket(m *GenesisMarketTaskResult) {
	if r.marketResult != nil {
		panic("Genesis Market State already set, developer error!!!")
	}
	r.marketResult = m
}

type GenesisMinerTaskResult struct {
	StateModel   *miner.MinerState
	PowerModel   *miner.MinerPower
	SectorModels miner.MinerSectorInfos
	DealModels   miner.MinerDealSectors
}

type GenesisMarketTaskResult struct {
	DealModels     market.MarketDealStates
	ProposalModesl market.MarketDealProposals
}
