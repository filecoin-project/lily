package gap

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/storage"
)

type Notifier struct {
	DB                   *storage.Database
	queue                *queue.AsynQ
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	tasks                []string
	done                 chan struct{}
}

func NewNotifier(node lens.API, db *storage.Database, queue *queue.AsynQ, name string, minHeight, maxHeight uint64, tasks []string) *Notifier {
	return &Notifier{
		DB:        db,
		queue:     queue,
		node:      node,
		name:      name,
		maxHeight: maxHeight,
		minHeight: minHeight,
		tasks:     tasks,
	}
}

func (g *Notifier) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	gaps, heights, err := g.DB.ConsolidateGaps(ctx, g.minHeight, g.maxHeight, g.tasks...)
	if err != nil {
		return err
	}

	idx := distributed.NewTipSetIndexer(g.queue)

	for _, height := range heights {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ts, err := g.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), types.EmptyTSK)
		if err != nil {
			return err
		}

		if success, err := idx.TipSet(ctx, ts, queue.FillQueue, gaps[height]...); err != nil {
			return err
		} else if !success {
			continue
		}
	}

	return nil
}

func (g *Notifier) Done() <-chan struct{} {
	return g.done
}
