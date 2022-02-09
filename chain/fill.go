package chain

import (
	"context"
	"github.com/filecoin-project/lily/lens/task"
	"sort"
	"time"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

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

func (g *GapFiller) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	gaps, heights, err := g.consolidateGaps(ctx)
	if err != nil {
		return err
	}
	fillLog := log.With("type", "gap_fill")
	fillStart := time.Now()
	fillLog.Infow("gap fill start", "start", fillStart.String(), "total_epoch_gaps", len(gaps), "from", g.minHeight, "to", g.maxHeight, "task", g.tasks)

	idx := 0
	for _, height := range heights {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		taskAPI, err := task.NewTaskAPI(g.node)
		if err != nil {
			return err
		}
		runStart := time.Now()
		indexer, err := NewTipSetIndexer(taskAPI, g.DB, 0, g.name, gaps[height])
		if err != nil {
			return err
		}

		// walk a single height at a time since there is no guarantee neighboring heights share the same missing tasks.
		if err := NewWalker(indexer, g.node, height, height).Run(ctx); err != nil {
			log.Errorw("fill failed", "height", height, "error", err.Error(), "tasks", gaps[height])
			// TODO we could add an error to the gap report in a follow on if needed, but the actualy error should
			// exist in the processing report if this fails.
			continue
		}
		fillLog.Infow("fill success", "epoch", height, "epochs_remaining", len(gaps)-idx, "tasks_filled", gaps[height], "duration", time.Since(runStart))

		if err := g.setGapsFilled(ctx, height, gaps[height]...); err != nil {
			return err
		}
		idx += 1
	}
	fillLog.Infow("gap fill complete", "duration", time.Since(fillStart), "total_epoch_gaps", len(gaps), "epoch_gaps_filled", idx, "from", g.minHeight, "to", g.maxHeight, "task", g.tasks)
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
