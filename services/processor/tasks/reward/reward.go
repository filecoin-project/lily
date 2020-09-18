package reward

import (
	"bytes"
	"context"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/reward"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	rewardmodel "github.com/filecoin-project/sentinel-visor/model/actors/reward"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessRewardTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessRewardTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("rewardtask")
		return next()
	})
	logging.SetLogLevel("rewardtask", "debug")
	// log all task
	pool.Middleware((*ProcessRewardTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessRewardTask).Task)

	return pool, queue
}

type ProcessRewardTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	ts        types.TipSetKey
	head      cid.Cid
	stateroot cid.Cid
}

func (p *ProcessRewardTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("starting process reward task", "job", job.ID, "args", job.Args)
	return next()
}

func (p *ProcessRewardTask) ParseArgs(job *work.Job) error {
	tsStr := job.ArgString("ts")
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

	stateroot, err := cid.Decode(srStr)
	if err != nil {
		return err
	}

	head, err := cid.Decode(headStr)
	if err != nil {
		return err
	}

	var tsKey types.TipSetKey
	if err := tsKey.UnmarshalJSON([]byte(tsStr)); err != nil {
		return err
	}

	p.ts = tsKey
	p.head = head
	p.stateroot = stateroot
	return nil
}

func (p *ProcessRewardTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}

	ctx := context.TODO()

	rewardStateRaw, err := p.node.ChainReadObj(ctx, p.head)
	if err != nil {
		return err
	}

	var rwdState reward.State
	if err := rwdState.UnmarshalCBOR(bytes.NewReader(rewardStateRaw)); err != nil {
		return err
	}

	p.pubCh <- &rewardmodel.ChainReward{
		StateRoot:                         p.stateroot.String(),
		CumSumBaseline:                    rwdState.CumsumBaseline.String(),
		CumSumRealized:                    rwdState.CumsumRealized.String(),
		EffectiveBaselinePower:            rwdState.EffectiveBaselinePower.String(),
		NewBaselinePower:                  rwdState.ThisEpochBaselinePower.String(),
		NewRewardSmoothedPositionEstimate: rwdState.ThisEpochRewardSmoothed.PositionEstimate.String(),
		NewRewardSmoothedVelocityEstimate: rwdState.ThisEpochRewardSmoothed.VelocityEstimate.String(),
		TotalMinedReward:                  rwdState.TotalMined.String(),
		NewReward:                         rwdState.ThisEpochReward.String(),
		EffectiveNetworkTime:              int64(rwdState.EffectiveNetworkTime),
	}

	return nil
}
