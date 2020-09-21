package init

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	initmodel "github.com/filecoin-project/sentinel-visor/model/actors/init"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessInitActorTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessInitActorTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("markettask")
		return next()
	})
	logging.SetLogLevel("markettask", "info")
	// log all task
	pool.Middleware((*ProcessInitActorTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessInitActorTask).Task)

	return pool, queue
}

type ProcessInitActorTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	head      cid.Cid
	stateroot cid.Cid
	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
}

func (p *ProcessInitActorTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("starting init actor task", "job", job.ID, "args", job.Args)
	return next()
}

func (p *ProcessInitActorTask) ParseArgs(job *work.Job) error {
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

	p.head = mhead
	p.tsKey = tsKey
	p.ptsKey = ptsKey
	p.stateroot = mstateroot
	return nil
}

func (p *ProcessInitActorTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}

	ctx := context.TODO()

	pred := state.NewStatePredicates(p.node)
	stateDiff := pred.OnInitActorChange(pred.OnAddressMapChange())
	changed, val, err := stateDiff(ctx, p.ptsKey, p.tsKey)
	if err != nil {
		return err
	}
	if !changed {
		return err
	}
	changes, ok := val.(*state.InitActorAddressChanges)
	if !ok {
		return fmt.Errorf("unknown type returned by init acotr hamt predicate: %T", val)
	}

	out := make(initmodel.IdAddressList, len(changes.Added)+len(changes.Modified))
	for idx, add := range changes.Added {
		out[idx] = &initmodel.IdAddress{
			ID:        add.ID.String(),
			Address:   add.PK.String(),
			StateRoot: p.stateroot.String(),
		}
	}
	for idx, mod := range changes.Modified {
		out[idx] = &initmodel.IdAddress{
			ID:        mod.To.ID.String(),
			Address:   mod.To.PK.String(),
			StateRoot: p.stateroot.String(),
		}
	}
	p.pubCh <- out
	return nil
}
