package genesis

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	init_ "github.com/filecoin-project/sentinel-visor/model/actors/init_"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/model/actors/multisig"
	"github.com/filecoin-project/sentinel-visor/model/actors/power"
)

// TODO delete me?
type ProcessGenesisSingletonResult struct {
	minerResults    miner.MinerTaskResultList
	msigResults     multisig.MultisigTaskResultList
	marketResult    *GenesisMarketTaskResult
	initActorResult *GenesisInitActorTaskResult
	powerResult     *power.PowerTaskResult
}

func (r *ProcessGenesisSingletonResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	// marshal miner actors
	if err := r.minerResults.Persist(ctx, s, version); err != nil {
		return xerrors.Errorf("persisting miner task result list: %w", err)
	}
	// marshal market actor
	if r.marketResult != nil {
		if err := r.marketResult.DealModels.Persist(ctx, s, version); err != nil {
			return err
		}
		if err := r.marketResult.ProposalModels.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	// marshal init actor
	if r.initActorResult != nil {
		if err := r.initActorResult.AddressMap.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	// marshal power actor
	if r.powerResult != nil {
		if err := r.powerResult.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	// marshal multisig actor
	if r.msigResults != nil {
		if err := r.msigResults.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}

func (r *ProcessGenesisSingletonResult) AddMsig(m *multisig.MultisigTaskResult) {
	r.msigResults = append(r.msigResults, m)
}

func (r *ProcessGenesisSingletonResult) AddMiner(m *miner.MinerTaskResult) {
	r.minerResults = append(r.minerResults, m)
}

func (r *ProcessGenesisSingletonResult) SetPower(p *power.PowerTaskResult) {
	if r.powerResult != nil {
		panic("Genesis Power State already set, developer error!!!")
	}
	r.powerResult = p
}

func (r *ProcessGenesisSingletonResult) SetMarket(m *GenesisMarketTaskResult) {
	if r.marketResult != nil {
		panic("Genesis Market State already set, developer error!!!")
	}
	r.marketResult = m
}

func (r *ProcessGenesisSingletonResult) SetInitActor(m *GenesisInitActorTaskResult) {
	if r.initActorResult != nil {
		panic("Genesis InitActor State already set, developer error!!!")
	}
	r.initActorResult = m
}

type GenesisMinerTaskResult struct {
	CurrDeadlineModels miner.MinerCurrentDeadlineInfoList
	FeeDebtModels      miner.MinerFeeDebtList
	LockedFundsModel   miner.MinerLockedFundsList
	InfoModels         miner.MinerInfoList
	SectorModels       miner.MinerSectorInfoList
	DealModels         miner.MinerSectorDealList
}

type GenesisMarketTaskResult struct {
	DealModels     market.MarketDealStates
	ProposalModels market.MarketDealProposals
}

func (g *GenesisMarketTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if g.DealModels != nil {
		if err := g.DealModels.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if g.ProposalModels != nil {
		if err := g.ProposalModels.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}

type GenesisInitActorTaskResult struct {
	AddressMap init_.IdAddressList
}

func (g *GenesisInitActorTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if g.AddressMap != nil {
		return g.AddressMap.Persist(ctx, s, version)
	}
	return nil
}
