package chain

import (
	"context"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/types"
)

func NewIndexingTipSetObserver(obs TipSetObserver, cache *TipSetCache) *IndexingTipSetObserver {
	return &IndexingTipSetObserver{
		obs:   obs,
		cache: cache,
	}
}

type IndexingTipSetObserver struct {
	obs   TipSetObserver
	cache *TipSetCache
}

func (i *IndexingTipSetObserver) Apply(ctx context.Context, ts *types.TipSet) error {
	log.Debugw("add tipset", "height", ts.Height(), "tipset", ts.Key().String())
	tail, err := i.cache.Add(ts)
	if err != nil {
		log.Errorw("tipset cache add", "error", err.Error())
	}

	// Send the tipset that fell out of the confidence window to the observer
	if tail != nil {
		return i.obs.TipSet(ctx, tail)
	}
	return nil
}

func (i *IndexingTipSetObserver) Revert(ctx context.Context, ts *types.TipSet) error {
	log.Debugw("revert tipset", "height", ts.Height(), "tipset", ts.Key().String())
	err := i.cache.Revert(ts)
	if err != nil {
		log.Errorw("tipset cache revert", "error", err.Error())
	}
	return nil
}

var _ events.TipSetObserver = (*IndexingTipSetObserver)(nil)
