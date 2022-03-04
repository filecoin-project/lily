package worker

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/queue"
)

var log = logging.Logger("lily/worker")

type IndexPriority string

const (
	High   IndexPriority = "High"
	Medium IndexPriority = "Medium"
	Low    IndexPriority = "Low"
)

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet, priority IndexPriority, tasks ...string) error
}

type RedisIndexer struct {
	Client *asynq.Client
}

func (p *RedisIndexer) TipSet(ctx context.Context, ts *types.TipSet, priority IndexPriority, tasks ...string) error {
	task, err := NewIndexTipSetTask(ts, tasks)
	if err != nil {
		return err
	}
	info, err := p.Client.EnqueueContext(ctx, task, asynq.Queue(string(priority)))
	if err != nil {
		return err
	}
	log.Infow("enqueued task", "id", info.ID, "queue", info.Queue, "tasks", tasks)
	return nil
}

func NewProducer(cfg *queue.RedisConfig) *RedisIndexer {
	redisCfg := asynq.RedisClientOpt{
		Network:  cfg.Network,
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}
	c := asynq.NewClient(
		redisCfg,
	)
	return &RedisIndexer{
		Client: c,
	}
}

const (
	TypeIndexTipSet = "tipset:index"
)

type IndexTipSetPayload struct {
	TipSet *types.TipSet
	Tasks  []string
}

func NewIndexTipSetTask(ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(IndexTipSetPayload{TipSet: ts, Tasks: tasks})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeIndexTipSet, payload), nil
}

func (p *RedisIndexer) TipSetWithTasks(ctx context.Context, ts *types.TipSet, tasks []string) error {
	task, err := NewIndexTipSetTask(ts, tasks)
	if err != nil {
		return err
	}
	info, err := p.Client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}
	log.Infow("enqueued task", "id", info.ID, "queue", info.Queue, "tasks", tasks)
	return nil
}

func (p *RedisIndexer) Close() error {
	return p.Client.Close()
}
