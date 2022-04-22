package queue

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/config"
)

var _ distributed.Queue = (*AsynQ)(nil)

type AsynQ struct {
	c *asynq.Client
}

func NewAsynQ(cfg config.AsynqRedisConfig) *AsynQ {
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Network:  cfg.Network,
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return &AsynQ{c: asynqClient}
}

func (r *AsynQ) EnqueueTs(ctx context.Context, ts *types.TipSet, priority string, taskNames ...string) error {
	var task *asynq.Task
	var err error
	if priority == FillQueue {
		task, err = tasks.NewGapFillTipSetTask(ts, taskNames)
		if err != nil {
			return err
		}
	} else {
		task, err = tasks.NewIndexTipSetTask(ts, taskNames)
		if err != nil {
			return err
		}
	}

	_, err = r.c.EnqueueContext(ctx, task, asynq.Queue(priority))
	if err != nil {
		return err
	}

	return nil

}
