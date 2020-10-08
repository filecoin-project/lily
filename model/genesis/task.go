package genesis

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	init_ "github.com/filecoin-project/sentinel-visor/model/actors/init"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

type ProcessGenesisSingletonResult struct {
	minerResults    []*GenesisMinerTaskResult
	marketResult    *GenesisMarketTaskResult
	initActorResult *GenesisInitActorTaskResult
}

func (r *ProcessGenesisSingletonResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "ProcessGenesisSingletonResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

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
			if err := r.marketResult.ProposalModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		if r.initActorResult != nil {
			if err := r.initActorResult.AddressMap.PersistWithTx(ctx, tx); err != nil {
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

func (r *ProcessGenesisSingletonResult) SetInitActor(m *GenesisInitActorTaskResult) {
	if r.initActorResult != nil {
		panic("Genesis InitActor State already set, developer error!!!")
	}
	r.initActorResult = m
}

type GenesisMinerTaskResult struct {
	StateModel   *miner.MinerState
	PowerModel   *miner.MinerPower
	SectorModels miner.MinerSectorInfos
	DealModels   miner.MinerDealSectors
}

type GenesisMarketTaskResult struct {
	DealModels     market.MarketDealStates
	ProposalModels market.MarketDealProposals
}

type GenesisInitActorTaskResult struct {
	AddressMap init_.IdAddressList
}
