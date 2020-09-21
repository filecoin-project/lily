package message

import (
	"context"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	messagemodel "github.com/filecoin-project/sentinel-visor/model/messages"
)

func Setup(concurrency uint, taskName, poolName string, redisPool *redis.Pool, node lens.API, pubCh chan<- model.Persistable) (*work.WorkerPool, *work.Enqueuer) {
	pool := work.NewWorkerPool(ProcessMessageTask{}, concurrency, poolName, redisPool)
	queue := work.NewEnqueuer(poolName, redisPool)

	// https://github.com/gocraft/work/issues/10#issuecomment-237580604
	// adding fields via a closure gives the workers access to the lotus api, a global could also be used here
	pool.Middleware(func(mt *ProcessMessageTask, job *work.Job, next work.NextMiddlewareFunc) error {
		mt.node = node
		mt.pubCh = pubCh
		mt.log = logging.Logger("messagetask")
		return next()
	})
	logging.SetLogLevel("messagetask", "info")
	// log all task
	pool.Middleware((*ProcessMessageTask).Log)

	// register task method and don't allow retying
	pool.JobWithOptions(taskName, work.JobOptions{
		MaxFails: 1,
	}, (*ProcessMessageTask).Task)

	return pool, queue
}

type ProcessMessageTask struct {
	node lens.API
	log  *logging.ZapEventLogger

	pubCh chan<- model.Persistable

	ts        types.TipSetKey
	stateroot cid.Cid
}

func (pm *ProcessMessageTask) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	pm.log.Infow("starting message task", "job", job.ID, "args", job.Args)
	return next()
}

func (pm *ProcessMessageTask) ParseArgs(job *work.Job) error {
	tsStr := job.ArgString("ts")
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

	var tsKey types.TipSetKey
	if err := tsKey.UnmarshalJSON([]byte(tsStr)); err != nil {
		return err
	}

	pm.ts = tsKey
	pm.stateroot = stateroot
	return nil
}

func (pm *ProcessMessageTask) Task(job *work.Job) error {
	if err := pm.ParseArgs(job); err != nil {
		return err
	}

	ctx := context.Background()
	ctx, span := global.Tracer("").Start(ctx, "ProcessMessageTask.Task")
	defer span.End()

	msgs, blkMsgs, err := pm.fetchMessages(ctx)
	if err != nil {
		return err
	}

	rcts, err := pm.fetchReceipts(ctx)
	if err != nil {
		return err
	}

	pm.pubCh <- &messagemodel.MessageTaskResult{
		Messages:      msgs,
		BlockMessages: blkMsgs,
		Receipts:      rcts,
	}
	return nil
}

func (pm *ProcessMessageTask) fetchMessages(ctx context.Context) (messagemodel.Messages, messagemodel.BlockMessages, error) {
	msgs := messagemodel.Messages{}
	bmsgs := messagemodel.BlockMessages{}
	msgsSeen := map[cid.Cid]struct{}{}

	// TODO consider performing this work in parallel.
	for _, blk := range pm.ts.Cids() {
		blkMsgs, err := pm.node.ChainGetBlockMessages(ctx, blk)
		if err != nil {
			return nil, nil, err
		}

		vmm := make([]*types.Message, 0, len(blkMsgs.Cids))
		for _, m := range blkMsgs.BlsMessages {
			vmm = append(vmm, m)
		}

		for _, m := range blkMsgs.SecpkMessages {
			vmm = append(vmm, &m.Message)
		}

		for _, message := range vmm {
			bmsgs = append(bmsgs, &messagemodel.BlockMessage{
				Block:   blk.String(),
				Message: message.Cid().String(),
			})

			// so we don't create duplicate message models.
			if _, seen := msgsSeen[message.Cid()]; seen {
				continue
			}

			var msgSize int
			if b, err := message.Serialize(); err == nil {
				msgSize = len(b)
			}
			msgs = append(msgs, &messagemodel.Message{
				Cid:        message.Cid().String(),
				From:       message.From.String(),
				To:         message.To.String(),
				Value:      message.Value.String(),
				GasFeeCap:  message.GasFeeCap.String(),
				GasPremium: message.GasPremium.String(),
				GasLimit:   message.GasLimit,
				SizeBytes:  msgSize,
				Nonce:      message.Nonce,
				Method:     uint64(message.Method),
				Params:     message.Params,
			})
			msgsSeen[message.Cid()] = struct{}{}
		}
	}
	return msgs, bmsgs, nil
}

func (pm *ProcessMessageTask) fetchReceipts(ctx context.Context) (messagemodel.Receipts, error) {
	out := messagemodel.Receipts{}

	for _, blk := range pm.ts.Cids() {
		recs, err := pm.node.ChainGetParentReceipts(ctx, blk)
		if err != nil {
			return nil, err
		}
		msgs, err := pm.node.ChainGetParentMessages(ctx, blk)
		if err != nil {
			return nil, err
		}

		for i, r := range recs {
			out = append(out, &messagemodel.Receipt{
				Message:   msgs[i].Cid.String(),
				StateRoot: pm.stateroot.String(),
				Idx:       i,
				ExitCode:  int64(r.ExitCode),
				GasUsed:   r.GasUsed,
				Return:    r.Return,
			})
		}
	}
	return out, nil
}
