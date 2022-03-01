package worker

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("lily/worker")

type Producer struct {
	Client *asynq.Client
}

func NewProducer(addr string) *Producer {
	c := asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
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
