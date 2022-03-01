package worker

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer"
)

type IndexHandler struct {
	im *indexer.Manager
}

func NewIndexHandler(m *indexer.Manager) *IndexHandler {
	return &IndexHandler{im: m}
}

func (ih *IndexHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("indexing tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	_, err := ih.im.TipSet(ctx, p.TipSet, p.Tasks...)
	if err != nil {
		return err
	}
	return nil
}
