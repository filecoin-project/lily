package queue

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/config"
)

var _ distributed.Queue = (*AsynQ)(nil)

type AsynQ struct {
	c *asynq.Client
}

func NewAsynq(cfg config.AsynqRedisConfig) *AsynQ {
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

func (r *AsynQ) EnqueueTipSet(ctx context.Context, ts *types.TipSet, indexType indexer.IndexerType, taskNames ...string) error {
	ctx, span := otel.Tracer("").Start(ctx, "AsnyQ.EnqueueTipSet")
	defer span.End()

	var task *asynq.Task
	var err error
	if indexType == indexer.Fill {
		task, err = tasks.NewGapFillTipSetTask(ctx, ts, taskNames)
		if err != nil {
			return err
		}
	} else {
		task, err = tasks.NewIndexTipSetTask(ctx, ts, taskNames)
		if err != nil {
			return err
		}
	}

	if span.IsRecording() {
		span.SetAttributes(attribute.String("task_type", task.Type()), attribute.StringSlice("tasks", taskNames), attribute.String("index_type", indexType.String()))
	}

	_, err = r.c.EnqueueContext(ctx, task, asynq.Queue(indexType.String()))
	if err != nil {
		return err
	}

	return nil

}
