package gap

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/integrated"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/storage"
)

var log = logging.Logger("lily/chain/gap")

type Filler struct {
	DB                   *storage.Database
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	tasks                []string
	done                 chan struct{}
}

func NewFiller(node lens.API, db *storage.Database, name string, minHeight, maxHeight uint64, tasks []string) *Filler {
	return &Filler{
		DB:        db,
		node:      node,
		name:      name,
		maxHeight: maxHeight,
		minHeight: minHeight,
		tasks:     tasks,
	}
}

func (g *Filler) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	gaps, heights, err := g.DB.ConsolidateGaps(ctx, g.minHeight, g.maxHeight, g.tasks...)
	if err != nil {
		return err
	}
	fillStart := time.Now()
	log.Infow("gap fill start", "start", fillStart.String(), "total_epoch_gaps", len(gaps), "from", g.minHeight, "to", g.maxHeight, "task", g.tasks, "reporter", g.name)

	taskAPI, err := datasource.NewDataSource(g.node)
	if err != nil {
		return err
	}

	index, err := integrated.NewManager(taskAPI, g.DB, g.name)
	if err != nil {
		return err
	}

	for _, height := range heights {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		runStart := time.Now()

		log.Infow("filling gap", "height", heights, "reporter", g.name)

		ts, err := g.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), types.EmptyTSK)
		if err != nil {
			return err
		}

		log.Infof("got tipset for height %d, tipset height %d", heights, ts.Height())
		if success, err := index.TipSet(ctx, ts, indexer.WithTasks(gaps[height])); err != nil {
			log.Errorw("fill indexing encountered fatal error", "height", height, "tipset", ts.Key().String(), "error", err, "tasks", gaps[height], "reporter", g.name)
			return err
		} else if !success {
			log.Errorw("fill indexing failed to successfully index tipset, skipping fill for tipset, gap remains", "height", height, "tipset", ts.Key().String(), "tasks", gaps[height], "reporter", g.name)
			continue
		}
		log.Infow("fill success", "epoch", ts.Height(), "tasks_filled", gaps[height], "duration", time.Since(runStart), "reporter", g.name)

		if err := g.DB.SetGapsFilled(ctx, height, gaps[height]...); err != nil {
			return err
		}
	}
	log.Infow("gap fill complete", "duration", time.Since(fillStart), "total_epoch_gaps", len(gaps), "from", g.minHeight, "to", g.maxHeight, "task", g.tasks, "reporter", g.name)

	return nil
}

func (g *Filler) Done() <-chan struct{} {
	return g.done
}
