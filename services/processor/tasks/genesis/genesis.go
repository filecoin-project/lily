package genesis

import (
	"bytes"
	"context"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"

	api "github.com/filecoin-project/sentinel-visor/lens/lotus"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
	genesismodel "github.com/filecoin-project/sentinel-visor/model/genesis"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node api.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
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
	logging.SetLogLevel("genesistask", "info")
	// log all task
	pool.Middleware((*ProcessGenesisSingletonTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessGenesisSingletonTask).Task)

	return pool, queue
}

type ProcessGenesisSingletonTask struct {
	node lapi.FullNode
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
	ctx := context.TODO()

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
			// TODO
		case builtin.CronActorCodeID:
			// TODO
		case builtin.AccountActorCodeID:
			// TODO
		case builtin.StoragePowerActorCodeID:
			// TODO
		case builtin.StorageMarketActorCodeID:
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
	store := api.NewAPIIpldStore(ctx, p.node)

	// actual miner actor state and miner info
	var mstate miner.State
	astb, err := p.node.ChainReadObj(ctx, act.Head)
	if err != nil {
		return nil, err
	}
	if err := mstate.UnmarshalCBOR(bytes.NewReader(astb)); err != nil {
		return nil, err
	}
	minfo, err := mstate.GetInfo(store)
	if err != nil {
		return nil, err
	}

	// miner raw and qual power
	// TODO this needs caching so we don't re-fetch the power actors claim table for every tipset.
	mpower, err := p.node.StateMinerPower(ctx, addr, p.genesis)
	if err != nil {
		return nil, err
	}

	msectors, err := p.node.StateMinerSectors(ctx, addr, nil, true, p.genesis)

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
		PeerID:     minfo.PeerId,
		SectorSize: minfo.SectorSize.ShortString(),
	}

	sectorsModel := make(minermodel.MinerSectorInfos, len(msectors))
	dealsModel := minermodel.MinerDealSectors{}
	for idx, sector := range msectors {
		sectorsModel[idx] = &minermodel.MinerSectorInfo{
			MinerID:               addr.String(),
			SectorID:              uint64(sector.ID),
			StateRoot:             p.stateroot.String(),
			SealedCID:             sector.Info.SealedCID.String(),
			ActivationEpoch:       int64(sector.Info.Activation),
			ExpirationEpoch:       int64(sector.Info.Expiration),
			DealWeight:            sector.Info.DealWeight.String(),
			VerifiedDealWeight:    sector.Info.VerifiedDealWeight.String(),
			InitialPledge:         sector.Info.InitialPledge.String(),
			ExpectedDayReward:     sector.Info.ExpectedDayReward.String(),
			ExpectedStoragePledge: sector.Info.ExpectedStoragePledge.String(),
		}
		for _, dealID := range sector.Info.DealIDs {
			dealsModel = append(dealsModel, &minermodel.MinerDealSector{
				MinerID:  addr.String(),
				SectorID: uint64(sector.ID),
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
