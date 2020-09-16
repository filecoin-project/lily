package market

import (
	"context"
	"golang.org/x/xerrors"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/lotus/chain/types"

	api "github.com/filecoin-project/sentinel-visor/lens/lotus"
	"github.com/filecoin-project/sentinel-visor/model"
	marketmodel "github.com/filecoin-project/sentinel-visor/model/actors/market"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node api.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessMarketTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessMarketTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("minertask")
		return next()
	})
	logging.SetLogLevel("markettask", "info")
	// log all task
	pool.Middleware((*ProcessMarketTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessMarketTask).Task)

	return pool, queue
}

type ProcessMarketTask struct {
	node lapi.FullNode
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	head      cid.Cid
	stateroot cid.Cid
	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
}

func (pmt *ProcessMarketTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	pmt.log.Infow("Starting Market Task", "name", job.Name, "Args", job.Args)
	return next()
}

func (pmt *ProcessMarketTask) ParseArgs(job *work.Job) error {
	headStr := job.ArgString("head")
	if err := job.ArgError(); err != nil {
		return err
	}

	srStr := job.ArgString("stateroot")
	if err := job.ArgError(); err != nil {
		return err
	}

	tsStr := job.ArgString("ts")
	if err := job.ArgError(); err != nil {
		return err
	}

	ptsStr := job.ArgString("pts")
	if err := job.ArgError(); err != nil {
		return err
	}

	mhead, err := cid.Decode(headStr)
	if err != nil {
		return err
	}

	mstateroot, err := cid.Decode(srStr)
	if err != nil {
		return err
	}

	var tsKey types.TipSetKey
	if err := tsKey.UnmarshalJSON([]byte(tsStr)); err != nil {
		return err
	}

	var ptsKey types.TipSetKey
	if err := ptsKey.UnmarshalJSON([]byte(ptsStr)); err != nil {
		return err
	}

	pmt.head = mhead
	pmt.tsKey = tsKey
	pmt.ptsKey = ptsKey
	pmt.stateroot = mstateroot
	return nil
}

func (pmt *ProcessMarketTask) Task(job *work.Job) error {
	if err := pmt.ParseArgs(job); err != nil {
		return err
	}

	ctx := context.TODO()

	proposals, err := pmt.marketDealProposalChanges(ctx)
	if err != nil {
		return err
	}

	states, err := pmt.marketDealStateChanges(ctx)
	if err != nil {
		return err
	}

	pmt.pubCh <- &marketmodel.MarketTaskResult{
		Proposals: proposals,
		States:    states,
	}

	return nil
}

func (pmt *ProcessMarketTask) marketDealStateChanges(ctx context.Context) (marketmodel.MarketDealStates, error) {
	pred := state.NewStatePredicates(pmt.node)
	stateDiff := pred.OnStorageMarketActorChanged(pred.OnDealStateChanged(pred.OnDealStateAmtChanged()))
	changed, val, err := stateDiff(ctx, pmt.ptsKey, pmt.tsKey)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, nil
	}
	changes, ok := val.(*state.MarketDealStateChanges)
	if !ok {
		// indicates a developer error or breaking change in lotus
		return nil, xerrors.Errorf("Unknown type returned by Deal State AMT predicate: %T", val)
	}
	out := make(marketmodel.MarketDealStates, len(changes.Added)+len(changes.Modified))
	for idx, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealState{
			DealID:           uint64(add.ID),
			StateRoot:        pmt.stateroot.String(),
			SectorStartEpoch: int64(add.Deal.SectorStartEpoch),
			LastUpdateEpoch:  int64(add.Deal.LastUpdatedEpoch),
			SlashEpoch:       int64(add.Deal.SlashEpoch),
		}
	}
	for idx, mod := range changes.Modified {
		out[idx] = &marketmodel.MarketDealState{
			DealID:           uint64(mod.ID),
			SectorStartEpoch: int64(mod.To.SectorStartEpoch),
			LastUpdateEpoch:  int64(mod.To.LastUpdatedEpoch),
			SlashEpoch:       int64(mod.To.SlashEpoch),
			StateRoot:        pmt.stateroot.String(),
		}
	}
	return out, nil
}

func (pmt *ProcessMarketTask) marketDealProposalChanges(ctx context.Context) (marketmodel.MarketDealProposals, error) {
	pred := state.NewStatePredicates(pmt.node)
	stateDiff := pred.OnStorageMarketActorChanged(pred.OnDealProposalChanged(pred.OnDealProposalAmtChanged()))
	changed, val, err := stateDiff(ctx, pmt.ptsKey, pmt.tsKey)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, nil
	}
	changes, ok := val.(*state.MarketDealProposalChanges)
	if !ok {
		// indicates a developer error or breaking change in lotus
		return nil, xerrors.Errorf("Unknown type returned by Deal Proposal AMT predicate: %T", val)
	}

	out := make(marketmodel.MarketDealProposals, len(changes.Added))

	for idx, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealProposal{
			DealID:               uint64(add.ID),
			StateRoot:            pmt.stateroot.String(),
			PaddedPieceSize:      uint64(add.Proposal.PieceSize),
			UnpaddedPieceSize:    uint64(add.Proposal.PieceSize.Unpadded()),
			StartEpoch:           int64(add.Proposal.StartEpoch),
			EndEpoch:             int64(add.Proposal.EndEpoch),
			ClientID:             add.Proposal.Client.String(),
			ProviderID:           add.Proposal.Provider.String(),
			ClientCollateral:     add.Proposal.ClientCollateral.String(),
			ProviderCollateral:   add.Proposal.ProviderCollateral.String(),
			StoragePricePerEpoch: add.Proposal.StoragePricePerEpoch.String(),
			PieceCID:             add.Proposal.PieceCID.String(),
			IsVerified:           add.Proposal.VerifiedDeal,
			Label:                add.Proposal.Label,
		}
	}
	return out, nil
}
