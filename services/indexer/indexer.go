package indexer

import (
	"container/list"
	"context"

	lotus_api "github.com/filecoin-project/lotus/api"
	pg "github.com/go-pg/pg/v10"
	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	store "github.com/filecoin-project/lotus/chain/store"
	types "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	storage "github.com/filecoin-project/sentinel-visor/storage"
)

var log = logging.Logger("indexer")

// TODO figure our if you want this or the init handler
func NewIndexer(s *storage.Database, n lens.API) *Indexer {
	return &Indexer{
		storage: s,
		node:    n,
	}
}

type Indexer struct {
	storage *storage.Database
	node    lens.API

	startingHeight int64
	startingBlock  cid.Cid
	genesis        *types.TipSet

	// TODO base this value on the spec: https://github.com/filecoin-project/specs-actors/pull/702
	finality int
}

// InitHandler initializes Indexer with state needed to start sycning head events
func (i *Indexer) InitHandler(ctx context.Context) error {
	gen, err := i.node.ChainGetGenesis(ctx)
	if err != nil {
		return xerrors.Errorf("get genesis: %w", err)
	}
	i.genesis = gen
	blk, height, err := i.mostRecentlySyncedBlockHeight(ctx)
	if err != nil {
		return xerrors.Errorf("get synced block height: %w", err)
	}

	finality := 1400
	i.startingBlock = blk
	i.startingHeight = height
	i.finality = finality

	log.Infow("initialized Indexer", "startingBlock", blk.String(), "startingHeight", height, "finality", finality)
	return nil
}

// Start runs the Indexer which blocks processing chain events until the context is cancelled or the api closes the
// connection.
func (i *Indexer) Start(ctx context.Context) error {
	log.Info("starting Indexer")
	hc, err := i.node.ChainNotify(ctx)
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
			if err := i.index(ctx, headEvents); err != nil {
				return xerrors.Errorf("index: %w", err)
			}
		}
	}
}

func (i *Indexer) index(ctx context.Context, headEvents []*lotus_api.HeadChange) error {
	ctx, span := global.Tracer("").Start(ctx, "Indexer.index")
	defer span.End()

	for _, head := range headEvents {
		log.Debugw("index", "event", head.Type)
		switch head.Type {
		case store.HCCurrent:
			fallthrough
		case store.HCApply:
			// collect all blocks to index starting from head and walking down the chain
			toIndex, err := i.collectBlocksToIndex(ctx, head.Val, i.startingHeight)
			if err != nil {
				return xerrors.Errorf("collect blocks: %w", err)
			}

			// if there are no new blocks short circuit
			if toIndex.Size() == 0 {
				return nil
			}

			// persist the blocks to storage
			if err := toIndex.Persist(ctx, i.storage.DB); err != nil {
				return xerrors.Errorf("persist: %w", err)
			}

			// keep the heights block we have seen so we don't recollect it.
			i.startingBlock, i.startingHeight = toIndex.Highest()
		case store.HCRevert:

			// TODO
		}
	}
	return nil
}

// Read Operations //

// TODO not sure if returning a map here is required, it gets passed to the publisher and then storage
// which doesn't need the CID key. I think we are just doing this for deduplication.
func (i *Indexer) collectBlocksToIndex(ctx context.Context, head *types.TipSet, maxHeight int64) (*UnindexedBlockData, error) {
	ctx, span := global.Tracer("").Start(ctx, "Indexer.CollectBlocks", trace.WithAttributes(label.Int64("height", int64(head.Height()))))
	defer span.End()

	// get at most finality blocks not exceeding maxHeight. These are blocks we have in the database but have not processed.
	// Now we are going to walk down the chain from `head` until we have visited all blocks not in the database.
	synced, err := i.storage.UnprocessedIndexedBlocks(ctx, int(maxHeight), i.finality)
	if err != nil {
		return nil, xerrors.Errorf("get unprocessed blocks: %w", err)
	}
	log.Infow("collect synced blocks", "count", len(synced))
	// well, this is complete shit
	has := make(map[cid.Cid]struct{})
	for _, c := range synced {
		key, err := cid.Decode(c.Cid)
		if err != nil {
			return nil, xerrors.Errorf("decode cid: %w", err)
		}
		has[key] = struct{}{}
	}
	// walk backwards from head until we find a block that we have

	toSync := NewUnindexedBlockData()
	toVisit := list.New()

	for _, header := range head.Blocks() {
		toVisit.PushBack(header)
	}

	for toVisit.Len() > 0 {
		bh := toVisit.Remove(toVisit.Back()).(*types.BlockHeader)
		_, has := has[bh.Cid()]
		if seen := toSync.Has(bh); seen || has {
			continue
		}

		toSync.Add(bh)

		if toSync.Size()%500 == 10 {
			log.Debugw("to visit", "toVisit", toVisit.Len(), "toSync", toSync.Size(), "current_height", bh.Height)
		}

		if bh.Height == 0 {
			continue
		}

		pts, err := i.node.ChainGetTipSet(ctx, types.NewTipSetKey(bh.Parents...))
		if err != nil {
			return nil, xerrors.Errorf("get tipset: %w", err)
		}

		for _, header := range pts.Blocks() {
			toVisit.PushBack(header)
		}
	}

	log.Debugw("collected unsynced blocks", "count", toSync.Size())
	return toSync, nil
}

func (i *Indexer) mostRecentlySyncedBlockHeight(ctx context.Context) (cid.Cid, int64, error) {
	ctx, span := global.Tracer("").Start(ctx, "Indexer.mostRecentlySyncedBlockHeight")
	defer span.End()

	task, err := i.storage.MostRecentSyncedBlock(ctx)
	if err != nil {
		if err == pg.ErrNoRows {
			return i.genesis.Cids()[0], 0, xerrors.Errorf("query recent synced: %w", err)
		}
		return cid.Undef, 0, err
	}
	c, err := cid.Decode(task.Cid)
	if err != nil {
		return cid.Undef, 0, xerrors.Errorf("decode cid: %w", err)
	}
	return c, task.Height, nil
}
