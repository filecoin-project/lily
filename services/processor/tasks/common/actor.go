package common

import (
	"context"
	"encoding/json"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"strconv"

	commonmodel "github.com/filecoin-project/sentinel-visor/model/actors/common"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessActorTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessActorTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("commonactortask")
		return next()
	})
	logging.SetLogLevel("commonactortask", "info")
	// log all task
	pool.Middleware((*ProcessActorTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessActorTask).Task)

	return pool, queue
}

type ProcessActorTask struct {
	node lapi.FullNode
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	tsKey     types.TipSetKey
	ptsKey    types.TipSetKey
	stateroot cid.Cid
	addr      address.Address
	head      cid.Cid
	code      cid.Cid
	balance   big.Int
	nonce     uint64
}

func (p *ProcessActorTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	p.log.Infow("starting common actor task", "job", job.ID, "args", job.Args)
	return next()
}

func (p *ProcessActorTask) ParseArgs(job *work.Job) error {
	// this needs a better pattern....
	tsStr := job.ArgString("ts")
	if err := job.ArgError(); err != nil {
		return err
	}

	ptsStr := job.ArgString("pts")
	if err := job.ArgError(); err != nil {
		return err
	}

	srStr := job.ArgString("stateroot")
	if err := job.ArgError(); err != nil {
		return err
	}

	addrStr := job.ArgString("address")
	if err := job.ArgError(); err != nil {
		return err
	}

	headStr := job.ArgString("head")
	if err := job.ArgError(); err != nil {
		return err
	}

	codeStr := job.ArgString("code")
	if err := job.ArgError(); err != nil {
		return err
	}

	balStr := job.ArgString("balance")
	if err := job.ArgError(); err != nil {
		return err
	}

	nonceStr := job.ArgString("nonce")
	if err := job.ArgError(); err != nil {
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

	stateroot, err := cid.Decode(srStr)
	if err != nil {
		return err
	}

	addr, err := address.NewFromString(addrStr)
	if err != nil {
		return err
	}

	head, err := cid.Decode(headStr)
	if err != nil {
		return err
	}

	code, err := cid.Decode(codeStr)
	if err != nil {
		return err
	}

	balance, err := big.FromString(balStr)
	if err != nil {
		return err
	}

	nonce, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		return err
	}

	p.tsKey = tsKey
	p.ptsKey = ptsKey
	p.stateroot = stateroot
	p.addr = addr
	p.head = head
	p.code = code
	p.balance = balance
	p.nonce = nonce
	return nil
}

func (p *ProcessActorTask) Task(job *work.Job) error {
	if err := p.ParseArgs(job); err != nil {
		return err
	}

	ctx := context.TODO()

	ast, err := p.node.StateReadState(ctx, p.addr, p.tsKey)
	if err != nil {
		return err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return err
	}

	p.pubCh <- &commonmodel.ActorTaskResult{
		Actor: &commonmodel.Actor{
			ID:        p.addr.String(),
			StateRoot: p.stateroot.String(),
			Code:      builtin.ActorNameByCode(p.code),
			Head:      p.head.String(),
			Balance:   p.balance.String(),
			Nonce:     p.nonce,
		},
		State: &commonmodel.ActorState{
			Head:  p.head.String(),
			Code:  p.code.String(),
			State: string(state),
		},
	}
	return nil
}
