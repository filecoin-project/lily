package indexer

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	lotus_api "github.com/filecoin-project/lotus/api"
	store "github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var log = logging.Logger("indexer")

// A TipSetObserver waits for notifications of new tipsets.
type TipSetObserver interface {
	TipSet(ctx context.Context, ts *types.TipSet) error
}

// NewChainHeadIndexer creates a new ChainHeadIndexer. confidence sets the number of tipsets that will be held
// in a cache awaiting possible reversion. Tipsets will be written to the database when they are evicted from
// the cache due to incoming later tipsets.
func NewChainHeadIndexer(obs TipSetObserver, opener lens.APIOpener, confidence int) *ChainHeadIndexer {
	return &ChainHeadIndexer{
		opener:     opener,
		obs:        obs,
		confidence: confidence,
		cache:      NewTipSetCache(confidence),
	}
}

// ChainHeadIndexer is a task that indexes blocks by following the chain head.
type ChainHeadIndexer struct {
	opener     lens.APIOpener
	obs        TipSetObserver
	confidence int          // size of tipset cache
	cache      *TipSetCache // caches tipsets for possible reversion
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *ChainHeadIndexer) Run(ctx context.Context) error {
	node, closer, err := c.opener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}
	defer closer()

	hc, err := node.ChainNotify(ctx)
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

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "indexheadblock"))

	for _, ch := range headEvents {
		switch ch.Type {
		case store.HCCurrent:
			log.Debugw("current tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			err := c.cache.SetCurrent(ch.Val)
			if err != nil {
				log.Errorw("tipset cache set current", "error", err.Error())
			}

			// If we have a zero confidence window then we need to notify every tipset we see
			if c.confidence == 0 {
				c.obs.TipSet(ctx, ch.Val)
			}
		case store.HCApply:
			log.Debugw("add tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			tail, err := c.cache.Add(ch.Val)
			if err != nil {
				log.Errorw("tipset cache add", "error", err.Error())
			}

			// Send the tipset that fell out of the confidence window to the observer
			if tail != nil {
				c.obs.TipSet(ctx, tail)
			}

		case store.HCRevert:
			log.Debugw("revert tipset", "height", ch.Val.Height(), "tipset", ch.Val.Key().String())
			err := c.cache.Revert(ch.Val)
			if err != nil {
				log.Errorw("tipset cache revert", "error", err.Error())
			}
		}
	}

	log.Debugw("tipset cache", "height", c.cache.Height(), "tail_height", c.cache.TailHeight(), "length", c.cache.Len())

	return nil
}

var _ TipSetObserver = (*TipSetBlockIndexer)(nil)

// A TipSetBlockIndexer waits for tipsets and persists their block data into a database.
type TipSetBlockIndexer struct {
	data    *UnindexedBlockData
	storage *storage.Database
}

func NewTipSetBlockIndexer(d *storage.Database) *TipSetBlockIndexer {
	return &TipSetBlockIndexer{
		data:    NewUnindexedBlockData(),
		storage: d,
	}
}

func (t *TipSetBlockIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
	t.data.AddTipSet(ts)
	if t.data.Size() > 0 {
		// persist the blocks to storage
		log.Debugw("persisting batch", "count", t.data.Size(), "height", t.data.Height())
		if err := t.data.Persist(ctx, t.storage.DB); err != nil {
			return xerrors.Errorf("persist: %w", err)
		}
	}

	return nil
}
