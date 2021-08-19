package gap

import (
	"context"
	"sort"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("visor/gap")

type GapFiller struct {
	DB                   *storage.Database
	opener               lens.APIOpener
	name                 string
	minHeight, maxHeight uint64
	tasks                []string
}

func NewGapFiller(o lens.APIOpener, db *storage.Database, name string, maxHeight, minHeight uint64, tasks []string) *GapFiller {
	return &GapFiller{
		DB:        db,
		opener:    o,
		name:      name,
		maxHeight: maxHeight,
		minHeight: minHeight,
		tasks:     tasks,
	}
}

func (g *GapFiller) Run(ctx context.Context) error {
	gaps, heights, err := g.consolidateGaps(ctx)
	if err != nil {
		return err
	}
	fillLog := log.With("type", "fill")
	fillLog.Infow("run", "count", len(gaps))

	idx := 0
	for _, height := range heights {
		indexer, err := chain.NewTipSetIndexer(g.opener, g.DB, 0, g.name, gaps[height])
		if err != nil {
			return err
		}

		// walk a single height at a time since there is no guarantee neighboring heights share the same missing tasks.
		if err := chain.NewWalker(indexer, g.opener, height, height).Run(ctx); err != nil {
			log.Errorw("fill failed", "height", height, "error", err.Error())
			// TODO we could add an error to the gap report in a follow on if needed, but the actualy error should
			// exist in the processing report if this fails.
			continue
		}
		fillLog.Infow("fill success", "height", height, "remaining", len(gaps)-idx)

		if err := g.setGapsFilled(ctx, height); err != nil {
			return err
		}
		idx += 1
	}
	return nil
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
func (g *GapFiller) setGapsFilled(ctx context.Context, height int64) error {
	if _, err := g.DB.AsORM().ModelContext(ctx, &visor.GapReport{}).
		Set("status = 'FILLED'").
		Where("height = ?", height).
		Update(); err != nil {
		return err
	}
	return nil
}
