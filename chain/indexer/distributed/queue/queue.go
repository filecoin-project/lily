package queue

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/config"
)

const (
	TypeIndexTipSet = "tipset:index"
)

type IndexTipSetPayload struct {
	TipSet   *types.TipSet
	Priority string
	Tasks    []string
}

func NewIndexTipSetTask(ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(IndexTipSetPayload{TipSet: ts, Tasks: tasks})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeIndexTipSet, payload), nil
}

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

func (r *AsynQ) EnqueueTs(ctx context.Context, ts *types.TipSet, priority string, tasks ...string) error {
	task, err := NewIndexTipSetTask(ts, tasks)
	if err != nil {
		return err
	}

	_, err = r.c.EnqueueContext(ctx, task, asynq.Queue(priority))
	if err != nil {
		return err
	}

	return nil

}
