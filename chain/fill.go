package chain

import (
	"context"
	"sort"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
)

var fillLog = logging.Logger("lily/chain/fill")

type GapFiller struct {
	DB                   *storage.Database
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	tasks                []string
	done                 chan struct{}
}

func NewGapFiller(node lens.API, db *storage.Database, name string, minHeight, maxHeight uint64, tasks []string) *GapFiller {
	return &GapFiller{
		DB:        db,
		node:      node,
		name:      name,
		maxHeight: maxHeight,
		minHeight: minHeight,
		tasks:     tasks,
	}
}

func (g *GapFiller) Close() {
	return
}

func (g *GapFiller) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	gaps, heights, err := g.consolidateGaps(ctx)
	if err != nil {
		return err
	}
	fillStart := time.Now()
	fillLog.Infow("gap fill start", "start", fillStart.String(), "total_epoch_gaps", len(gaps), "from", g.minHeight, "to", g.maxHeight, "task", g.tasks)

	taskAPI, err := datasource.NewDataSource(g.node)
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
		index := indexer.NewManager(taskAPI, g.DB, g.name, gaps[height])
		if err := index.Init(); err != nil {
			return err
		}

		fillLog.Infof("filling height %d", heights)
		ts, err := g.node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), types.EmptyTSK)
		if err != nil {
			return err
		}
		fillLog.Infof("got tipset for height %d, tipset height %d", heights, ts.Height())
		if success, err := index.TipSet(ctx, ts); err != nil {
			fillLog.Errorw("fill indexing encountered fatal error", "height", height, "tipset", ts.Key().String(), "error", err, "tasks", gaps[height])
			return err
		} else if !success {
			fillLog.Errorw("fill indexing failed to successfully index tipset, skipping fill for tipset, gap remains", "height", height, "tipset", ts.Key().String(), "tasks", gaps[height])
			continue
		}
		fillLog.Infow("fill success", "epoch", ts.Height(), "tasks_filled", gaps[height], "duration", time.Since(runStart))

		if err := g.setGapsFilled(ctx, height, gaps[height]...); err != nil {
			return err
		}
	}
	fillLog.Infow("gap fill complete", "duration", time.Since(fillStart), "total_epoch_gaps", len(gaps), "from", g.minHeight, "to", g.maxHeight, "task", g.tasks)

	return nil
}

func (g *GapFiller) Done() <-chan struct{} {
	return g.done
}

// returns a map of heights to missing tasks, and a list of heights to iterate the map in order with.
func (g *GapFiller) consolidateGaps(ctx context.Context) (map[int64][]string, []int64, error) {
	gaps, err := g.queryGaps(ctx)
	if err != nil {
		return nil, nil, err
	}
	// used to walk gaps in order, should help optimize some caching.
	heights := make([]int64, 0, len(gaps))
	out := make(map[int64][]string)
	for _, gap := range gaps {
		if _, ok := out[gap.Height]; !ok {
			heights = append(heights, gap.Height)
		}
		out[gap.Height] = append(out[gap.Height], gap.Task)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})
	return out, heights, nil
}

func (g *GapFiller) queryGaps(ctx context.Context) ([]*visor.GapReport, error) {
	var out []*visor.GapReport
	if len(g.tasks) != 0 {
		if err := g.DB.AsORM().ModelContext(ctx, &out).
			Order("height desc").
			Where("status = ?", "GAP").
			Where("task = ANY (?)", pg.Array(g.tasks)).
			Where("height >= ?", g.minHeight).
			Where("height <= ?", g.maxHeight).
			Select(); err != nil {
			return nil, xerrors.Errorf("querying gap reports: %w", err)
		}
	} else {
		if err := g.DB.AsORM().ModelContext(ctx, &out).
			Order("height desc").
			Where("status = ?", "GAP").
			Where("height >= ?", g.minHeight).
			Where("height <= ?", g.maxHeight).
			Select(); err != nil {
			return nil, xerrors.Errorf("querying gap reports: %w", err)
		}
	}
	return out, nil
}

// mark all gaps at height as filled.
func (g *GapFiller) setGapsFilled(ctx context.Context, height int64, tasks ...string) error {
	if _, err := g.DB.AsORM().ModelContext(ctx, &visor.GapReport{}).
		Set("status = 'FILLED'").
		Where("height = ?", height).
		Where("task = ANY (?)", pg.Array(tasks)).
		Update(); err != nil {
		return err
	}
	return nil
}
