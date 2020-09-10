package services

import (
	"container/list"
	"context"
	"github.com/filecoin-project/visor/model/blocks"

	pg "github.com/go-pg/pg/v10"
	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	trace "go.opentelemetry.io/otel/api/trace"

	api "github.com/filecoin-project/lotus/api"
	store "github.com/filecoin-project/lotus/chain/store"
	types "github.com/filecoin-project/lotus/chain/types"
	storage "github.com/filecoin-project/visor/storage"
)

var log = logging.Logger("indexer")

// TODO figure our if you want this or the init handler
func NewIndexer(s *storage.Database, n api.FullNode) *Indexer {
	return &Indexer{
		storage: s,
		node:    n,
	}
}

type Indexer struct {
	storage *storage.Database
	node    api.FullNode

	tracer trace.Tracer

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

// Start runs the Indexer and can be aborted by context cancellation
func (i *Indexer) Start(ctx context.Context) {
	log.Info("starting Indexer")
	hc, err := i.node.ChainNotify(ctx)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping Indexer")
				return
			case headEvents, ok := <-hc:
				if !ok {
					log.Warn("ChainNotify channel closed")
					return
				}
				if err := i.index(ctx, headEvents); err != nil {
					panic(err)
				}
			}
		}
	}()
}

func (i *Indexer) index(ctx context.Context, headEvents []*api.HeadChange) error {
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

// TODO put this somewhere else, maybe in the model?
type UnindexedBlockData struct {
	has     map[cid.Cid]struct{}
	highest *types.BlockHeader

	blks              blocks.BlockHeaders
	synced            blocks.BlocksSynced
	parents           blocks.BlockParents
	drandEntries      blocks.DrandEntries
	drandBlockEntries blocks.DrandBlockEntries
}

func (u *UnindexedBlockData) Highest() (cid.Cid, int64) {
	return u.highest.Cid(), int64(u.highest.Height)
}

func (u *UnindexedBlockData) Add(bh *types.BlockHeader) {
	u.has[bh.Cid()] = struct{}{}

	if u.highest == nil {
		u.highest = bh
	} else if u.highest.Height < bh.Height {
		u.highest = bh
	}

	u.blks = append(u.blks, blocks.NewBlockHeader(bh))
	u.synced = append(u.synced, blocks.NewBlockSynced(bh))
	u.parents = append(u.parents, blocks.NewBlockParents(bh)...)
	u.drandEntries = append(u.drandEntries, blocks.NewDrandEnties(bh)...)
	u.drandBlockEntries = append(u.drandBlockEntries, blocks.NewDrandBlockEntries(bh)...)
}

func (u *UnindexedBlockData) Has(bh *types.BlockHeader) bool {
	_, has := u.has[bh.Cid()]
	return has
}

func (u *UnindexedBlockData) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Close(); err != nil {
			log.Errorw("closing unsynced block data transaction", "error", err.Error())
		}
	}()

	if err := u.blks.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if err := u.synced.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if err := u.parents.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if err := u.drandEntries.PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if err := u.drandBlockEntries.PersistWithTx(ctx, tx); err != nil {
		return err
	}

	return tx.CommitContext(ctx)
}

func (u *UnindexedBlockData) Size() int {
	return len(u.has)
}

// Read Operations //

// TODO not sure if returning a map here is required, it gets passed to the publisher and then storage
// which doesn't need the CID key. I think we are just doing this for deduplication.
func (i *Indexer) collectBlocksToIndex(ctx context.Context, head *types.TipSet, maxHeight int64) (*UnindexedBlockData, error) {
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

	toSync := &UnindexedBlockData{}
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

		if len(toSync.blks)%500 == 10 {
			log.Debugw("to visit", "toVisit", toVisit.Len(), "toSync", len(toSync.blks), "current_height", bh.Height)
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

	log.Debugw("collected unsynced blocks", "count", len(toSync.blks))
	return toSync, nil
}

func (i *Indexer) mostRecentlySyncedBlockHeight(ctx context.Context) (cid.Cid, int64, error) {
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
