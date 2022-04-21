package gap

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
)

type Finder struct {
	DB                   *storage.Database
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	tasks                []string
	done                 chan struct{}
}

func NewFinder(node lens.API, db *storage.Database, name string, minHeight, maxHeight uint64, tasks []string) *Finder {
	return &Finder{
		DB:        db,
		node:      node,
		name:      name,
		tasks:     tasks,
		maxHeight: maxHeight,
		minHeight: minHeight,
	}
}

type TaskHeight struct {
	Task   string
	Height uint64
	Status string
}

func (g *Finder) Find(ctx context.Context) (visor.GapReportList, error) {
	log.Debug("finding task epoch gaps")
	start := time.Now()
	var result []TaskHeight
	out := visor.GapReportList{}
	// returns a list of tasks and heights for all incomplete heights
	// and incomplete height is a height with less than len(tasks) entries, the tasks returned
	// are the completed tasks for the height, we can diff them against all known tasks to find the
	// missing ones.
	res, err := g.DB.AsORM().QueryContext(
		ctx,
		&result,
		`
SELECT * FROM gap_find(?,?,?,?,?);
`,
		pg.Array(g.tasks), // arg 0
		g.minHeight,       // arg 1
		g.maxHeight,       // arg 2
		visor.ProcessingStatusInformationNullRound, // arg 3
		visor.ProcessingStatusOK,                   // arg 4
	)
	if err != nil {
		return nil, err
	}
	log.Infow("executed find task epoch gap query", "count", res.RowsReturned(), "reporter", g.name)
	for _, th := range result {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		out = append(out, &visor.GapReport{
			Height:     int64(th.Height),
			Task:       th.Task,
			Status:     "GAP",
			Reporter:   g.name,
			ReportedAt: start,
		})
	}
	return out, nil
}

func (g *Finder) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	head, err := g.node.ChainHead(ctx)
	if err != nil {
		return err
	}
	if uint64(head.Height()) < g.maxHeight {
		return xerrors.Errorf("cannot look for gaps beyond chain head height %d", head.Height())
	}

	gaps, err := g.Find(ctx)
	if err != nil {
		return err
	}

	return g.DB.PersistBatch(ctx, gaps)
}

func (g *Finder) Done() <-chan struct{} {
	return g.done
}
