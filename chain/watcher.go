package chain

import (
	"context"

	lotus_api "github.com/filecoin-project/lotus/api"
	store "github.com/filecoin-project/lotus/chain/store"
	"go.opencensus.io/stats"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
)

// NewWatcher creates a new Watcher. confidence sets the number of tipsets that will be held
// in a cache awaiting possible reversion. Tipsets will be written to the database when they are evicted from
// the cache due to incoming later tipsets.
func NewWatcher(obs TipSetObserver, opener lens.APIOpener, confidence int) *Watcher {
	return &Watcher{
		opener:     opener,
		obs:        obs,
		confidence: confidence,
		cache:      NewTipSetCache(confidence),
	}
}

// Watcher is a task that indexes blocks by following the chain head.
type Watcher struct {
	opener     lens.APIOpener
	obs        TipSetObserver
	confidence int          // size of tipset cache
	cache      *TipSetCache // caches tipsets for possible reversion
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *Watcher) Run(ctx context.Context) error {
	node, closer, err := c.opener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}

	defer func() {
		closer()
		if err := c.obs.Close(); err != nil {
			log.Errorw("watcher failed to close TipSetObserver", "error", err)
		}
	}()

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

func (c *Watcher) index(ctx context.Context, headEvents []*lotus_api.HeadChange) error {
	ctx, span := global.Tracer("").Start(ctx, "Watcher.index")
	defer span.End()

	for _, ch := range headEvents {
		stats.Record(ctx, metrics.WatchHeight.M(int64(ch.Val.Height())))

		switch ch.Type {
		case store.HCCurrent:
			err := c.cache.SetCurrent(ch.Val)
			if err != nil {
				log.Errorw("tipset cache set current", "error", err.Error())
			}

			// If we have a zero confidence window then we need to notify every tipset we see
			if c.confidence == 0 {
				if err := c.obs.TipSet(ctx, ch.Val); err != nil {
					return xerrors.Errorf("notify tipset: %w", err)
				}
			}
		case store.HCApply:
			tail, err := c.cache.Add(ch.Val)
			if err != nil {
				log.Errorw("tipset cache add", "error", err.Error())
			}

			// Send the tipset that fell out of the confidence window to the observer
			if tail != nil {
				if err := c.obs.TipSet(ctx, tail); err != nil {
					return xerrors.Errorf("notify tipset: %w", err)
				}
			}

		case store.HCRevert:
			err := c.cache.Revert(ch.Val)
			if err != nil {
				log.Errorw("tipset cache revert", "error", err.Error())
			}
		}
	}

	log.Debugw("tipset cache", "height", c.cache.Height(), "tail_height", c.cache.TailHeight(), "length", c.cache.Len())

	return nil
}
