package genesis

import (
	"context"
	"strconv"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	init_ "github.com/filecoin-project/lotus/chain/actors/builtin/init"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	initmodel "github.com/filecoin-project/sentinel-visor/model/actors/init"
	marketmodel "github.com/filecoin-project/sentinel-visor/model/actors/market"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
	genesismodel "github.com/filecoin-project/sentinel-visor/model/genesis"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessGenesisSingletonTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessGenesisSingletonTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("genesistask")
		return next()
	})
	// log all task
	pool.Middleware((*ProcessGenesisSingletonTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessGenesisSingletonTask).Task)

	return pool, queue
}

type ProcessGenesisSingletonTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	genesis   types.TipSetKey
	stateroot cid.Cid
}

func (p *ProcessGenesisSingletonTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("Starting Job", "name", job.Name, "Args", job.Args)
	return next()
}

func (p *ProcessGenesisSingletonTask) ParseArgs(job *work.Job) error {
	srStr := job.ArgString("stateroot")
	if err := job.ArgError(); err != nil {
		return err
	}

	tsStr := job.ArgString("ts")
	if err := job.ArgError(); err != nil {
		return err
	}

	sr, err := cid.Decode(srStr)
	if err != nil {
		return err
	}

	var tsKey types.TipSetKey
	if err := tsKey.UnmarshalJSON([]byte(tsStr)); err != nil {
		return err
	}
	p.genesis = tsKey
	p.stateroot = sr
	return nil
}

func (p *ProcessGenesisSingletonTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}
	ctx := context.Background()
	ctx, span := global.Tracer("").Start(ctx, "ProcessGenesisSingletonTask.Task")
	defer span.End()

	genesisAddrs, err := p.node.StateListActors(ctx, p.genesis)
	if err != nil {
		return err
	}

	result := &genesismodel.ProcessGenesisSingletonResult{}
	for _, addr := range genesisAddrs {
		genesisAct, err := p.node.StateGetActor(ctx, addr, p.genesis)
		if err != nil {
			return err
		}
		switch genesisAct.Code {
		case builtin.SystemActorCodeID:
			// TODO
		case builtin.InitActorCodeID:
			res, err := p.initActorState(genesisAct)
			if err != nil {
				return err
			}
			result.SetInitActor(res)
		case builtin.CronActorCodeID:
			// TODO
		case builtin.AccountActorCodeID:
			// TODO
		case builtin.StoragePowerActorCodeID:
			// TODO
		case builtin.StorageMarketActorCodeID:
			res, err := p.storageMarketState(ctx)
			if err != nil {
				return err
			}
			result.SetMarket(res)
		case builtin.StorageMinerActorCodeID:
			res, err := p.storageMinerState(ctx, addr, genesisAct)
			if err != nil {
				return err
			}
			result.AddMiner(res)
		case builtin.PaymentChannelActorCodeID:
			// TODO
		case builtin.MultisigActorCodeID:
			// TODO
		case builtin.RewardActorCodeID:
			// TODO
		case builtin.VerifiedRegistryActorCodeID:
			// TODO
		default:
			p.log.Warnf("unknown actor in genesis state. address: %s code: %s head: %s", addr, genesisAct.Code, genesisAct.Head)
		}
	}
	p.pubCh <- result
	return nil
}

func (p *ProcessGenesisSingletonTask) storageMinerState(ctx context.Context, addr address.Address, act *types.Actor) (*genesismodel.GenesisMinerTaskResult, error) {
	// actual miner actor state and miner info
	mstate, err := miner.Load(p.node.Store(), act)
	if err != nil {
		return nil, err
	}

	// miner raw and qual power
	// TODO this needs caching so we don't re-fetch the power actors claim table for every tipset.
	mpower, err := p.node.StateMinerPower(ctx, addr, p.genesis)
	if err != nil {
		return nil, err
	}

	msectors, err := mstate.LoadSectors(nil)
	if err != nil {
		return nil, err
	}

	minfo, err := mstate.Info()
	if err != nil {
		return nil, err
	}

	powerModel := &minermodel.MinerPower{
		MinerID:              addr.String(),
		StateRoot:            p.stateroot.String(),
		RawBytePower:         mpower.MinerPower.RawBytePower.String(),
		QualityAdjustedPower: mpower.MinerPower.QualityAdjPower.String(),
	}

	stateModel := &minermodel.MinerState{
		MinerID:    addr.String(),
		OwnerID:    minfo.Owner.String(),
		WorkerID:   minfo.Worker.String(),
		PeerID:     minfo.PeerId.String(),
		SectorSize: minfo.SectorSize.ShortString(),
	}

	sectorsModel := make(minermodel.MinerSectorInfos, len(msectors))
	dealsModel := minermodel.MinerDealSectors{}
	for idx, sector := range msectors {
		sectorsModel[idx] = &minermodel.MinerSectorInfo{
			MinerID:               addr.String(),
			SectorID:              uint64(sector.SectorNumber),
			StateRoot:             p.stateroot.String(),
			SealedCID:             sector.SealedCID.String(),
			ActivationEpoch:       int64(sector.Activation),
			ExpirationEpoch:       int64(sector.Expiration),
			DealWeight:            sector.DealWeight.String(),
			VerifiedDealWeight:    sector.VerifiedDealWeight.String(),
			InitialPledge:         sector.InitialPledge.String(),
			ExpectedDayReward:     sector.ExpectedDayReward.String(),
			ExpectedStoragePledge: sector.ExpectedStoragePledge.String(),
		}
		for _, dealID := range sector.DealIDs {
			dealsModel = append(dealsModel, &minermodel.MinerDealSector{
				MinerID:  addr.String(),
				SectorID: uint64(sector.SectorNumber),
				DealID:   uint64(dealID),
			})
		}
	}
	return &genesismodel.GenesisMinerTaskResult{
		StateModel:   stateModel,
		PowerModel:   powerModel,
		SectorModels: sectorsModel,
		DealModels:   dealsModel,
	}, nil
}

func (p *ProcessGenesisSingletonTask) initActorState(act *types.Actor) (*genesismodel.GenesisInitActorTaskResult, error) {
	initActorState, err := init_.Load(p.node.Store(), act)
	if err != nil {
		return nil, err
	}

	out := initmodel.IdAddressList{}
	if err := initActorState.ForEachActor(func(id abi.ActorID, addr address.Address) error {
		out = append(out, &initmodel.IdAddress{
			ID:        id.String(),
			Address:   addr.String(),
			StateRoot: p.stateroot.String(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return &genesismodel.GenesisInitActorTaskResult{AddressMap: out}, nil
}

func (p *ProcessGenesisSingletonTask) storageMarketState(ctx context.Context) (*genesismodel.GenesisMarketTaskResult, error) {
	dealStates, err := p.node.StateMarketDeals(ctx, p.genesis)
	if err != nil {
		return nil, err
	}

	states := make(marketmodel.MarketDealStates, len(dealStates))
	proposals := make(marketmodel.MarketDealProposals, len(dealStates))
	idx := 0
	for idStr, deal := range dealStates {
		dealID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, err
		}
		states[idx] = &marketmodel.MarketDealState{
			DealID:           dealID,
			SectorStartEpoch: int64(deal.State.SectorStartEpoch),
			LastUpdateEpoch:  int64(deal.State.LastUpdatedEpoch),
			SlashEpoch:       int64(deal.State.SlashEpoch),
			StateRoot:        p.stateroot.String(),
		}
		proposals[idx] = &marketmodel.MarketDealProposal{
			DealID:               dealID,
			StateRoot:            p.stateroot.String(),
			PaddedPieceSize:      uint64(deal.Proposal.PieceSize),
			UnpaddedPieceSize:    uint64(deal.Proposal.PieceSize.Unpadded()),
			StartEpoch:           int64(deal.Proposal.StartEpoch),
			EndEpoch:             int64(deal.Proposal.EndEpoch),
			ClientID:             deal.Proposal.Client.String(),
			ProviderID:           deal.Proposal.Provider.String(),
			ClientCollateral:     deal.Proposal.ClientCollateral.String(),
			ProviderCollateral:   deal.Proposal.ProviderCollateral.String(),
			StoragePricePerEpoch: deal.Proposal.StoragePricePerEpoch.String(),
			PieceCID:             deal.Proposal.PieceCID.String(),
			IsVerified:           deal.Proposal.VerifiedDeal,
			Label:                deal.Proposal.Label,
		}
		idx++
	}
	return &genesismodel.GenesisMarketTaskResult{
		states,
		proposals,
	}, nil
}
