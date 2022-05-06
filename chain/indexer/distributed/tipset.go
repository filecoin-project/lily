package distributed

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer"
)

var _ indexer.Indexer = (*TipSetIndexer)(nil)

type Queue interface {
	EnqueueTipSet(ctx context.Context, ts *types.TipSet, indexType indexer.IndexerType, tasks ...string) error
}

type TipSetIndexer struct {
	q Queue
}

func NewTipSetIndexer(q Queue) *TipSetIndexer {
	return &TipSetIndexer{q: q}
}

func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet, opts ...indexer.Option) (bool, error) {
	o, err := indexer.ConstructOptions(opts...)
	if err != nil {
		return false, err
	}
	if o.IndexType == indexer.Undefined {
		return false, fmt.Errorf("indexer type required")
	}
	log.Infow("index tipset", "height", ts.Height(), "type", o.IndexType.String(), "tasks", o.Tasks)
	if err := t.q.EnqueueTipSet(ctx, ts, o.IndexType, o.Tasks...); err != nil {
		return false, err
	}
	return true, nil
}
