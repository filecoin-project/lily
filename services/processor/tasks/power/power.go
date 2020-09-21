package power

import (
	"bytes"
	"context"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"

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
	logging.SetLogLevel("minertask", "info")
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

func (ppt *ProcessPowerTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	ppt.log.Infow("Starting job", "name", job.Name, "args", job.Args)
	return next()
}

func (ppt *ProcessPowerTask) ParseArgs(job *work.Job) error {
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

	ppt.maddr = maddr
	ppt.head = mhead
	ppt.tsKey = tsKey
	ppt.ptsKey = ptsKey
	ppt.stateroot = mstateroot
	return nil
}

func (ppt *ProcessPowerTask) Task(job *work.Job) error {
	if err := ppt.ParseArgs(job); err != nil {
		return err
	}
	ctx := context.Background()
	ctx, span := global.Tracer("").Start(ctx, "ProcessPowerTask.Task")
	defer span.End()

	powerActor, err := ppt.node.StateGetActor(ctx, builtin.StoragePowerActorAddr, ppt.tsKey)
	if err != nil {
		return err
	}

	powerStateRaw, err := ppt.node.ChainReadObj(ctx, powerActor.Head)
	if err != nil {
		return err
	}

	var powerActorState power.State
	if err := powerActorState.UnmarshalCBOR(bytes.NewReader(powerStateRaw)); err != nil {
		return err
	}

	ppt.pubCh <- &powermodel.ChainPower{
		StateRoot:                  ppt.stateroot.String(),
		NewRawBytesPower:           powerActorState.ThisEpochRawBytePower.String(),
		NewQABytesPower:            powerActorState.ThisEpochQualityAdjPower.String(),
		NewPledgeCollateral:        powerActorState.ThisEpochPledgeCollateral.String(),
		TotalRawBytesPower:         powerActorState.TotalRawBytePower.String(),
		TotalRawBytesCommitted:     powerActorState.TotalBytesCommitted.String(),
		TotalQABytesPower:          powerActorState.TotalQualityAdjPower.String(),
		TotalQABytesCommitted:      powerActorState.TotalQABytesCommitted.String(),
		TotalPledgeCollateral:      powerActorState.TotalPledgeCollateral.String(),
		QASmoothedPositionEstimate: powerActorState.ThisEpochQAPowerSmoothed.PositionEstimate.String(),
		QASmoothedVelocityEstimate: powerActorState.ThisEpochQAPowerSmoothed.VelocityEstimate.String(),
		MinerCount:                 powerActorState.MinerCount,
		MinimumConsensusMinerCount: powerActorState.MinerAboveMinPowerCount,
	}
	return nil
}
