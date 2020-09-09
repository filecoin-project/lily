package services

import (
	"container/list"
	"context"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/visor/storage"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/whyrusleeping/pubsub"
	"go.opentelemetry.io/otel/api/trace"
)

// TODO figure our if you want this or the init handler
func NewIndexer(s *storage.Database, p *Publisher, n api.FullNode) *Indexer {
	return &Indexer{
		storage: s,
		node:    n,
		pub:     p,
		log:     logging.Logger("visor/services/indexer"),
	}
}

type Indexer struct {
	storage *storage.Database
	node    api.FullNode
	pub     *Publisher

	events *pubsub.PubSub

	log    *logging.ZapEventLogger
	tracer trace.Tracer

	startingHeight int64
	startingBlock  cid.Cid
	genesis        *types.TipSet

	// TODO base this value on the spec: https://github.com/filecoin-project/specs-actors/pull/702
	finality int
}

// InitHandler initializes Indexer with state needed to start sycning head events
func (i *Indexer) InitHandler(ctx context.Context, eventSub *pubsub.PubSub) error {
	logging.SetLogLevel("*", "debug")

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

	i.events = eventSub

	i.log.Infow("initialized Indexer", "startingBlock", blk.String(), "startingHeight", height, "finality", finality)
	return nil
}

// Start runs the Indexer and can be aborted by context cancellation
func (i *Indexer) Start(ctx context.Context) {
	i.log.Info("starting Indexer")
	hc, err := i.node.ChainNotify(ctx)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				i.log.Info("stopping Indexer")
				return
			case headEvents, ok := <-hc:
				if !ok {
					i.log.Warn("ChainNotify channel closed")
					return
				}
				// NB: this is where we could us a worker pool of size one.
				if err := i.sync(ctx, headEvents); err != nil {
					panic(err)
				}
			}
		}
	}()
}

func (i *Indexer) sync(ctx context.Context, headEvents []*api.HeadChange) error {
	for _, event := range headEvents {
		i.log.Debugw("sync", "event", event.Type)
		switch event.Type {
		case store.HCCurrent:
			fallthrough
		case store.HCApply:
			tosync, err := i.collectUnsyncedBlocks(ctx, event.Val, i.startingHeight)
			if err != nil {
				return err
			}

			if len(tosync) == 0 {
				return nil
			}

			if err := i.pub.Publish(ctx, BlockHeaderPayload{
				headers: tosync,
				task:    true,
			}); err != nil {
				return err
			}

			i.startingBlock, i.startingHeight, err = i.mostRecentlySyncedBlockHeight(ctx)
			if err != nil {
				return err
			}
		case store.HCRevert:

			// TODO
		}
	}
	return nil
}

// Read Operations //

// TODO not sure if returning a map here is required, it gets passed to the publisher and then storage
// which doesn't need the CID key. I think we are just doing this for deduplication.
func (i *Indexer) collectUnsyncedBlocks(ctx context.Context, head *types.TipSet, maxHeight int64) (map[cid.Cid]*types.BlockHeader, error) {
	// get at most finality blocks not exceeding maxHeight. These are blocks we have in the database but have not processed.
	// Now we are going to walk down the chain from `head` until we have visited all blocks not in the database.
	synced, err := i.storage.IncompleteBlockProcessTasks(ctx, int(maxHeight), i.finality)
	if err != nil {
		return nil, err
	}
	i.log.Infow("collect synced blocks", "count", len(synced))
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

	toSync := map[cid.Cid]*types.BlockHeader{}
	toVisit := list.New()

	for _, header := range head.Blocks() {
		toVisit.PushBack(header)
	}

	for toVisit.Len() > 0 {
		bh := toVisit.Remove(toVisit.Back()).(*types.BlockHeader)
		_, has := has[bh.Cid()]
		if _, seen := toSync[bh.Cid()]; seen || has {
			continue
		}

		toSync[bh.Cid()] = bh
		if len(toSync)%500 == 10 {
			i.log.Debugw("to visit", "toVisit", toVisit.Len(), "toSync", len(toSync), "current_height", bh.Height)
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

	i.log.Debugw("collected unsynced blocks", "count", len(toSync))
	return toSync, nil
}

func (i *Indexer) mostRecentlySyncedBlockHeight(ctx context.Context) (cid.Cid, int64, error) {
	task, err := i.storage.MostRecentCompletedBlockProcessTask(ctx)
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
