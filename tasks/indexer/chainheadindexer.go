package indexer

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	lotus_api "github.com/filecoin-project/lotus/api"
	store "github.com/filecoin-project/lotus/chain/store"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var log = logging.Logger("indexer")

// NewChainHeadIndexer creates a new ChainHeadIndexer. confidence sets the number of tipsets that will be held
// in a cache awaiting possible reversion. Tipsets will be written to the database when they are evicted from
// the cache due to incoming later tipsets.
func NewChainHeadIndexer(d *storage.Database, node lens.API, confidence int) *ChainHeadIndexer {
	return &ChainHeadIndexer{
		node:       node,
		storage:    d,
		confidence: confidence,
		cache:      NewTipSetCache(confidence),
	}
}

// ChainHeadIndexer is a task that indexes blocks by following the chain head.
type ChainHeadIndexer struct {
	node       lens.API
	storage    *storage.Database
	confidence int          // size of tipset cache
	cache      *TipSetCache // caches tipsets for possible reversion
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *ChainHeadIndexer) Run(ctx context.Context) error {
	hc, err := c.node.ChainNotify(ctx)
	if err != nil {
		return xerrors.Errorf("chain notify: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case headEvents, ok := <-hc:
			if !ok {
				log.Warn("ChainNotify channel closed, stopping Indexer")
				return nil
			}
			if err := c.index(ctx, headEvents); err != nil {
				return xerrors.Errorf("index: %w", err)
			}
		}
	}
}

func (c *ChainHeadIndexer) index(ctx context.Context, headEvents []*lotus_api.HeadChange) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainHeadIndexer.index")
	defer span.End()

	data := NewUnindexedBlockData()

	var height int64
	for _, ch := range headEvents {
		switch ch.Type {
		case store.HCCurrent:
			fallthrough
		case store.HCApply:
			log.Debugw("add tipset", "event", ch.Type, "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			if int64(ch.Val.Height()) > height {
				height = int64(ch.Val.Height())
			}
			tail, err := c.cache.Add(ch.Val)
			if err != nil {
				log.Errorw("tipset cache", "error", err.Error())
			}

			// Send the tipset that fell out of the confidence window to the database
			if tail != nil {
				data.AddTipSet(tail)
			}

		case store.HCRevert:
			log.Debugw("revert tipset", "event", ch.Type, "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			err := c.cache.Revert(ch.Val)
			if err != nil {
				log.Errorw("tipset cache", "error", err.Error())
			}
		}
	}

	if data.Size() > 0 {
		// persist the blocks to storage
		log.Debugw("persisting batch", "count", data.Size(), "current_height", height)
		if err := data.Persist(ctx, c.storage.DB); err != nil {
			return xerrors.Errorf("persist: %w", err)
		}
	}
	return nil

}
