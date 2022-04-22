package tasks

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/storage"
)

const (
	TypeGapFillTipSet = "tipset:gapfill"
)

type GapFillTipSetPayload struct {
	TipSet *types.TipSet
	Tasks  []string
}

func NewGapFillTipSetTask(ts *types.TipSet, tasks []string) (*asynq.Task, error) {
	payload, err := json.Marshal(GapFillTipSetPayload{TipSet: ts, Tasks: tasks})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGapFillTipSet, payload), nil
}

type AsynqGapFillTipSetTaskHandler struct {
	indexer indexer.Indexer
	db      *storage.Database
}

func NewGapFillHandler(indexer indexer.Indexer, db *storage.Database) *AsynqGapFillTipSetTaskHandler {
	return &AsynqGapFillTipSetTaskHandler{indexer: indexer, db: db}
}

func (gh *AsynqGapFillTipSetTaskHandler) HandleGapFillTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p GapFillTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("gap fill tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	success, err := gh.indexer.TipSet(ctx, p.TipSet, "", p.Tasks...)
	if err != nil {
		return err
	}
	if !success {
		// TODO do we return an error here and try again or do we give up: depends on error, if no state, give up
		log.Warnw("failed to gap fill task successfully", "height", p.TipSet.Height(), "tipset", p.TipSet.Key().String())
	} else {
		if err := gh.db.SetGapsFilled(ctx, int64(p.TipSet.Height()), p.Tasks...); err != nil {
			return err
		}
	}
	return nil
}
