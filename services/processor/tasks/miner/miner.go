package miner

import (
	"context"
	"fmt"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessMinerTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessMinerTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("minertask")
		return next()
	})
	// log all task
	pool.Middleware((*ProcessMinerTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessMinerTask).Task)

	return pool, queue
}

type ProcessMinerTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	maddr     address.Address
	head      cid.Cid
	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
	stateroot cid.Cid
}

func (p *ProcessMinerTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("Starting Miner Task", "name", job.Name, "Args", job.Args)
	return next()
}

func (p *ProcessMinerTask) ParseArgs(job *work.Job) error {
	addrStr := job.ArgString("address")
	if err := job.ArgError(); err != nil {
		return err
	}

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

	maddr, err := address.NewFromString(addrStr)
	if err != nil {
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

	p.maddr = maddr
	p.head = mhead
	p.tsKey = tsKey
	p.ptsKey = ptsKey
	p.stateroot = mstateroot
	return nil
}

func (p *ProcessMinerTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}
	// TODO:
	// - all processing below can and probably should be done in parallel.
	// - processing is incomplete, see below TODO about sector inspection.
	// - need caching infront of the lotus api to avoid refetching power for same tipset.
	ctx := context.Background()
	ctx, span := global.Tracer("").Start(ctx, "ProcessMinerTask.Task")
	defer span.End()

	curActor, err := p.node.StateGetActor(ctx, p.maddr, p.tsKey)
	if err != nil {
		return xerrors.Errorf("loading current miner actor: %w", err)
	}

	curState, err := miner.Load(p.node.Store(), curActor)
	if err != nil {
		return xerrors.Errorf("loading current miner state: %w", err)
	}

	minfo, err := curState.Info()
	if err != nil {
		return xerrors.Errorf("loading miner info: %w", err)
	}

	// miner raw and qual power
	// TODO this needs caching so we don't re-fetch the power actors claim table (that holds this info) for every tipset.
	minerPower, err := p.node.StateMinerPower(ctx, p.maddr, p.tsKey)
	if err != nil {
		return xerrors.Errorf("loading miner power: %w", err)
	}

	// needed for diffing.
	prevActor, err := p.node.StateGetActor(ctx, p.maddr, p.ptsKey)
	if err != nil {
		return xerrors.Errorf("loading previous miner actor: %w", err)
	}

	prevState, err := miner.Load(p.node.Store(), prevActor)
	if err != nil {
		return xerrors.Errorf("loading previous miner actor state: %w", err)
	}

	preCommitChanges, err := miner.DiffPreCommits(prevState, curState)
	if err != nil {
		return xerrors.Errorf("diffing miner precommits: %w", err)
	}

	sectorChanges, err := miner.DiffSectors(prevState, curState)
	if err != nil {
		return xerrors.Errorf("diffing miner sectors: %w", err)
	}

	// miner partition changes
	partitionsDiff, err := p.minerPartitionsDiff(ctx, prevState, curState)
	if err != nil {
		return fmt.Errorf("diffing miner partitions: %v", err)
	}

	// TODO we still need to do a little bit more processing here around sectors to get all the info we need, but this is okay for spike.

	p.pubCh <- &minermodel.MinerTaskResult{
		Ts:               p.tsKey,
		Pts:              p.ptsKey,
		Addr:             p.maddr,
		StateRoot:        p.stateroot,
		Actor:            curActor,
		State:            curState,
		Info:             minfo,
		Power:            minerPower,
		PreCommitChanges: preCommitChanges,
		SectorChanges:    sectorChanges,
		PartitionDiff:    partitionsDiff,
	}
	return nil
}

func (p *ProcessMinerTask) minerPartitionsDiff(ctx context.Context, prevState, curState miner.State) (map[uint64]*minermodel.PartitionStatus, error) {
	panic("NYI")
}
