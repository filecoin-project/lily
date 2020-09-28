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

func NewChainHeadIndexer(d *storage.Database, node lens.API) *ChainHeadIndexer {
	return &ChainHeadIndexer{
		node:    node,
		storage: d,
	}
}

// ChainHeadIndexer is a task that indexes blocks by following the chain head.
type ChainHeadIndexer struct {
	node    lens.API
	storage *storage.Database
}

// Run starts following the chain head and blocks until the context is done or
// an error occurs.
func (c *ChainHeadIndexer) Run(ctx context.Context) error {
	log.Info("starting chain head indexer")
	hc, err := c.node.ChainNotify(ctx)
	if err != nil {
		return xerrors.Errorf("chain notify: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping Indexer")
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
	for _, head := range headEvents {
		log.Debugw("index", "event", head.Type)
		switch head.Type {
		case store.HCCurrent:
			fallthrough
		case store.HCApply:
			data.AddTipSet(head.Val)
			if int64(head.Val.Height()) > height {
				height = int64(head.Val.Height())
			}
		case store.HCRevert:
			// TODO
		}
	}

	// persist the blocks to storage
	log.Debugw("persisting batch", "count", data.Size(), "current_height", height)
	if err := data.Persist(ctx, c.storage.DB); err != nil {
		return xerrors.Errorf("persist: %w", err)
	}

	return nil

}
