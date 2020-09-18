package indexer

import (
	"container/list"
	"context"
	lotus_api "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/sentinel-visor/lens"
	pg "github.com/go-pg/pg/v10"
	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/opentracing/opentracing-go"

	store "github.com/filecoin-project/lotus/chain/store"
	types "github.com/filecoin-project/lotus/chain/types"
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
	if err := logging.SetLogLevel("*", "debug"); err != nil {
		return err
	}

	gen, err := i.node.ChainGetGenesis(ctx)
	if err != nil {
		return err
	}
	i.genesis = gen
	blk, height, err := i.mostRecentlySyncedBlockHeight(ctx)
	if err != nil {
		return err
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
		return err
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
				return err
			}
		}
	}
}

func (i *Indexer) index(ctx context.Context, headEvents []*lotus_api.HeadChange) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Indexer.index")
	defer span.Finish()

	for _, head := range headEvents {
		log.Debugw("index", "event", head.Type)
		switch head.Type {
		case store.HCCurrent:
			fallthrough
		case store.HCApply:
			// collect all blocks to index starting from head and walking down the chain
			toIndex, err := i.collectBlocksToIndex(ctx, head.Val, i.startingHeight)
			if err != nil {
				return err
			}

			// if there are no new blocks short circuit
			if toIndex.Size() == 0 {
				return nil
			}

			// persist the blocks to storage
			if err := toIndex.Persist(ctx, i.storage.DB); err != nil {
				return err
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "Indexer.CollectBlocks", opentracing.Tags{"height": head.Height})
	defer span.Finish()

	// get at most finality blocks not exceeding maxHeight. These are blocks we have in the database but have not processed.
	// Now we are going to walk down the chain from `head` until we have visited all blocks not in the database.
	synced, err := i.storage.UnprocessedIndexedBlocks(ctx, int(maxHeight), i.finality)
	if err != nil {
		return nil, err
	}
	log.Infow("collect synced blocks", "count", len(synced))
	// well, this is complete shit
	has := make(map[cid.Cid]struct{})
	for _, c := range synced {
		key, err := cid.Decode(c.Cid)
		if err != nil {
			return nil, err
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
			return nil, err
		}

		for _, header := range pts.Blocks() {
			toVisit.PushBack(header)
		}
	}

	log.Debugw("collected unsynced blocks", "count", toSync.Size())
	return toSync, nil
}

func (i *Indexer) mostRecentlySyncedBlockHeight(ctx context.Context) (cid.Cid, int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Indexer.mostRecentlySyncedBlockHeight")
	defer span.Finish()

	task, err := i.storage.MostRecentProcessedBlock(ctx)
	if err != nil {
		if err == pg.ErrNoRows {
			return i.genesis.Cids()[0], 0, nil
		}
		return cid.Undef, 0, err
	}
	c, err := cid.Decode(task.Cid)
	if err != nil {
		panic(err)
	}
	return c, task.Height, nil
}
