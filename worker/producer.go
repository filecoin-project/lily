package worker

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer"
)

var log = logging.Logger("lily/worker")

type Producer struct {
	Client *asynq.Client
}

func (p *Producer) TipSetAsync(ctx context.Context, ts *types.TipSet) error {
	return p.TipSetWithTasks(ctx, ts, indexer.AllTasks)
}

type RedisConfig struct {
	// Network type to use, either tcp or unix.
	// Default is tcp.
	Network string
	// Redis server address in "host:port" format.
	Addr string
	// Username to authenticate the current connection when Redis ACLs are used.
	// See: https://redis.io/commands/auth.
	Username string
	// Password to authenticate the current connection.
	// See: https://redis.io/commands/auth.
	Password string
	// Redis DB to select after connecting to a server.
	// See: https://redis.io/commands/select.
	DB int
	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
}

func NewProducer(cfg *RedisConfig) *Producer {
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
	return &Producer{
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

func (p *Producer) TipSetWithTasks(ctx context.Context, ts *types.TipSet, tasks []string) error {
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

func (p *Producer) Close() error {
	return p.Client.Close()
}
