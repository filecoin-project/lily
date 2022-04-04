package chain

import (
	"context"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

type GapIndexer struct {
	DB                   *storage.Database
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	taskSet              mapset.Set
	done                 chan struct{}
}

var FullTaskSet mapset.Set

func init() {
	FullTaskSet = mapset.NewSet()
	for _, t := range AllTasks {
		FullTaskSet.Add(t)
	}
}

func NewGapIndexer(node lens.API, db *storage.Database, name string, minHeight, maxHeight uint64, tasks []string) *GapIndexer {
	taskSet := mapset.NewSet()
	for _, t := range tasks {
		taskSet.Add(t)
	}
	return &GapIndexer{
		DB:        db,
		node:      node,
		name:      name,
		taskSet:   taskSet,
		maxHeight: maxHeight,
		minHeight: minHeight,
	}
}

func (g *GapIndexer) Run(ctx context.Context) error {
	// init the done channel for each run since jobs may be started and stopped.
	g.done = make(chan struct{})
	defer close(g.done)

	startTime := time.Now()

	head, err := g.node.ChainHead(ctx)
	if err != nil {
		return err
	}
	if uint64(head.Height()) < g.maxHeight {
		return xerrors.Errorf("cannot look for gaps beyond chain head height %d", head.Height())
	}

	findLog := log.With("type", "find")

	// looks for incomplete epochs. An incomplete epoch has some, but not all tasks in the processing report table.
	taskGaps, err := g.findTaskEpochGaps(ctx)
	if err != nil {
		return xerrors.Errorf("finding task epoch gaps: %w", err)
	}
	findLog.Infow("found gaps in tasks", "count", len(taskGaps))

	// looks for missing epochs and null rounds. A missing epoch is a non-null-round height missing from the processing report table
	heightGaps, nulls, err := g.findEpochGapsAndNullRounds(ctx, g.node)
	if err != nil {
		return xerrors.Errorf("finding epoch gaps: %w", err)
	}
	findLog.Infow("found gaps in epochs", "count", len(heightGaps))
	findLog.Infow("found null rounds", "count", len(nulls))

	// looks for entriest in the lily processing report table that have been skipped.
	skipGaps, err := g.findEpochSkips(ctx)
	if err != nil {
		return xerrors.Errorf("detecting skipped gaps: %w", err)
	}
	findLog.Infow("found skipped epochs", "count", len(skipGaps))

	var nullRounds visor.ProcessingReportList
	for _, epoch := range nulls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		nullRounds = append(nullRounds, &visor.ProcessingReport{
			Height:            int64(epoch),
			StateRoot:         "NULL_ROUND", // let gap fill add the correct state root for this epoch when it runs the consensus task.
			Reporter:          g.name,
			Task:              "gap_find",
			StartedAt:         startTime,
			CompletedAt:       time.Now(),
			Status:            visor.ProcessingStatusInfo,
			StatusInformation: visor.ProcessingStatusInformationNullRound, // the gap finding logic uses this value to exclude null rounds from gap report.
		})
	}

	return g.DB.PersistBatch(ctx, skipGaps, heightGaps, taskGaps, nullRounds)
}

func (g *GapIndexer) Done() <-chan struct{} {
	return g.done
}

type GapIndexerLens interface {
	ChainGetTipSetByHeight(ctx context.Context, epoch abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error)
}

func (g *GapIndexer) findEpochSkips(ctx context.Context) (visor.GapReportList, error) {
	log.Debug("finding skipped epochs")
	reportTime := time.Now()

	var skippedReports []visor.ProcessingReport
	if err := g.DB.AsORM().ModelContext(ctx, &skippedReports).
		Order("height desc").
		Where("status = ?", visor.ProcessingStatusSkip).
		Where("height >= ?", g.minHeight).
		Where("height <= ?", g.maxHeight).
		Select(); err != nil {
		return nil, xerrors.Errorf("query processing report skips: %w", err)
	}
	log.Debugw("executed find skipped epoch query", "count", len(skippedReports))

	gapReport := make([]*visor.GapReport, len(skippedReports))
	for idx, r := range skippedReports {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		gapReport[idx] = &visor.GapReport{
			Height:     r.Height,
			Task:       r.Task,
			Status:     "GAP",
			Reporter:   g.name,
			ReportedAt: reportTime,
		}
	}
	return gapReport, nil
}

func (g *GapIndexer) findEpochGapsAndNullRounds(ctx context.Context, node GapIndexerLens) (visor.GapReportList, []abi.ChainEpoch, error) {
	log.Debug("finding epoch gaps and null rounds")
	reportTime := time.Now()

	var nullRounds []abi.ChainEpoch
	var missingHeights []uint64
	res, err := g.DB.AsORM().QueryContext(
		ctx,
		&missingHeights,
		`
		SELECT s.i AS missing_epoch
		FROM generate_series(?, ?) s(i)
		WHERE NOT EXISTS (SELECT 1 FROM visor_processing_reports WHERE height = s.i AND status = ?)
		;
		`,
		g.minHeight, g.maxHeight, visor.ProcessingStatusOK)
	if err != nil {
		return nil, nil, err
	}
	log.Debugw("executed find epoch gap query", "count", res.RowsReturned())

	gapReport := make([]*visor.GapReport, 0, len(missingHeights))
	// walk the possible gaps and query lotus to determine if gap was a null round or missed epoch.
	for _, gap := range missingHeights {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}
		gh := abi.ChainEpoch(gap)
		tsgap, err := node.ChainGetTipSetByHeight(ctx, gh, types.EmptyTSK)
		if err != nil {
			return nil, nil, xerrors.Errorf("getting tipset by height %d: %w", gh, err)
		}
		if tsgap.Height() == gh {
			log.Debugw("found gap", "height", gh)
			for _, task := range AllTasks {
				gapReport = append(gapReport, &visor.GapReport{
					Height:     int64(tsgap.Height()),
					Task:       task,
					Status:     "GAP",
					Reporter:   g.name,
					ReportedAt: reportTime,
				})
			}
		} else {
			log.Debugw("found null round", "height", gh)
			nullRounds = append(nullRounds, gh)
		}
	}
	return gapReport, nullRounds, nil
}

type TaskHeight struct {
	Task   string
	Height uint64
	Status string
}

// TODO rather than use the length of `tasks` to determine where gaps are, use the contents to look for
// gaps in specific task. Forrest' SQL-Foo isn't good enough for this yet.
func (g *GapIndexer) findTaskEpochGaps(ctx context.Context) (visor.GapReportList, error) {
	log.Debug("finding task epoch gaps")
	start := time.Now()
	var result []TaskHeight
	var out visor.GapReportList
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
		pg.Array(AllTasks), // arg 0
		g.minHeight,        // arg 1
		g.maxHeight,        // arg 2
		visor.ProcessingStatusInformationNullRound, // arg 3
		visor.ProcessingStatusOK,                   // arg 4
	)
	if err != nil {
		return nil, err
	}
	log.Debugw("executed find task epoch gap query", "count", res.RowsReturned())
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
