package tasks

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer"
)

var log = logging.Logger("lily/queue/tasks")

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

type AsynqTipSetTaskHandler struct {
	indexer indexer.Indexer
}

func NewIndexHandler(i indexer.Indexer) *AsynqTipSetTaskHandler {
	return &AsynqTipSetTaskHandler{indexer: i}
}

func (ih *AsynqTipSetTaskHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("indexing tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	success, err := ih.indexer.TipSet(ctx, p.TipSet, "", p.Tasks...)
	if err != nil {
		return err
	}
	if !success {
		log.Warnw("failed to index task successfully", "height", p.TipSet.Height(), "tipset", p.TipSet.Key().String())
	}
	return nil
}
