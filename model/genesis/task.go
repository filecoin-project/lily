package genesis

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
	init_ "github.com/filecoin-project/sentinel-visor/model/actors/init"
	"github.com/filecoin-project/sentinel-visor/model/actors/market"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/model/actors/power"
)

type ProcessGenesisSingletonResult struct {
	minerResults    miner.MinerTaskResultList
	marketResult    *GenesisMarketTaskResult
	initActorResult *GenesisInitActorTaskResult
	powerResult     *power.PowerTaskResult
}

func (r *ProcessGenesisSingletonResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "ProcessGenesisSingletonResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		// persist miner actors
		if err := r.minerResults.PersistWithTx(ctx, tx); err != nil {
			return xerrors.Errorf("persisting miner task result list: %w", err)
		}
		// persist market actor
		if r.marketResult != nil {
			if err := r.marketResult.DealModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
			if err := r.marketResult.ProposalModels.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		// persist init actor
		if r.initActorResult != nil {
			if err := r.initActorResult.AddressMap.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		// persist power actor
		if r.powerResult != nil {
			if err := r.powerResult.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
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

type GenesisInitActorTaskResult struct {
	AddressMap init_.IdAddressList
}
