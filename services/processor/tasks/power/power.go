package power

import (
	"context"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	powermodel "github.com/filecoin-project/sentinel-visor/model/actors/power"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessPowerTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessPowerTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("minertask")
		return next()
	})
	// log all task
	pool.Middleware((*ProcessPowerTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessPowerTask).Task)

	return pool, queue
}

type ProcessPowerTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	maddr     address.Address
	head      cid.Cid
	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
	stateroot cid.Cid
}

func (p *ProcessPowerTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("Starting job", "name", job.Name, "args", job.Args)
	return next()
}

func (p *ProcessPowerTask) ParseArgs(job *work.Job) error {
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

func (p *ProcessPowerTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}
	ctx := context.Background()
	ctx, span := global.Tracer("").Start(ctx, "ProcessPowerTask.Task")
	defer span.End()

	powerActor, err := p.node.StateGetActor(ctx, builtin.StoragePowerActorAddr, p.tsKey)
	if err != nil {
		return xerrors.Errorf("loading power actor: %w", err)
	}

	pstate, err := power.Load(p.node.Store(), powerActor)
	if err != nil {
		return xerrors.Errorf("loading power actor state: %w", err)
	}

	locked, err := pstate.TotalLocked()
	if err != nil {
		return err
	}
	pow, err := pstate.TotalPower()
	if err != nil {
		return err
	}
	commit, err := pstate.TotalCommitted()
	if err != nil {
		return err
	}
	smoothed, err := pstate.TotalPowerSmoothed()
	if err != nil {
		return err
	}
	participating, total, err := pstate.MinerCounts()
	if err != nil {
		return err
	}

	p.pubCh <- &powermodel.ChainPower{
		StateRoot:                  p.stateroot.String(),
		TotalRawBytesPower:         pow.RawBytePower.String(),
		TotalQABytesPower:          pow.QualityAdjPower.String(),
		TotalRawBytesCommitted:     commit.RawBytePower.String(),
		TotalQABytesCommitted:      commit.QualityAdjPower.String(),
		TotalPledgeCollateral:      locked.String(),
		QASmoothedPositionEstimate: smoothed.PositionEstimate.String(),
		QASmoothedVelocityEstimate: smoothed.VelocityEstimate.String(),
		MinerCount:                 total,
		ParticipatingMinerCount:    participating,
	}
	return nil
}
