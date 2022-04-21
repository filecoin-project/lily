package distributed

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer"
)

var _ indexer.Indexer = (*TipSetIndexer)(nil)

type Queue interface {
	EnqueueTs(ctx context.Context, ts *types.TipSet, priority string, tasks ...string) error
}

type TipSetIndexer struct {
	q Queue
}

func NewTipSetIndexer(q Queue) *TipSetIndexer {
	return &TipSetIndexer{q: q}
}

func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet, priority string, tasks ...string) (bool, error) {
	log.Infow("Distributed Index Tipset", "height", ts.Height(), "priority", priority, "tasks", tasks)
	if err := t.q.EnqueueTs(ctx, ts, priority, tasks...); err != nil {
		return false, err
	}
	return true, nil
}
