package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

type GapIndexer struct {
	DB                   *storage.Database
	node                 lens.API
	name                 string
	minHeight, maxHeight uint64
	taskSet              mapset.Set
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
		WHERE NOT EXISTS (SELECT 1 FROM visor_processing_reports WHERE height = s.i);
		`,
		g.minHeight, g.maxHeight)
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

func (g *GapIndexer) findTaskEpochGaps(ctx context.Context) (visor.GapReportList, error) {
	log.Debug("finding task epoch gaps")
	start := time.Now()
	var result []TaskHeight
	var out visor.GapReportList
	var sqlFmtTaskValues []string
	for t := range g.taskSet.Iter() {
		sqlFmtTaskValues = append(sqlFmtTaskValues, fmt.Sprintf("('%s')", t))
	}

	// returns a list of tasks and heights for all incomplete heights and incomplete height
	// is a height without an 'OK' or 'NULL_ROUND' for g.tasks. Returned values indicate
	// heights and tasks which need to be filled.
	query := fmt.Sprintf(`
with

-- generate all heights in range
interesting_heights as (
	select *
	from generate_series(
		?0,
		?1
	)
	as x(height)
)
,

-- enum all tasks for which we want to find gaps
interesting_tasks as (
	select *
	from (values %s
		-- example values in sqlFmtTasks:
		-- ('actorstatesraw'),
		-- ('actorstatespower'),
		-- ('actorstatesreward'),
		-- ('actorstatesminer'),
		-- ('actorstatesinit'),
		-- ('actorstatesmarket'),
		-- ('actorstatesmultisig'),
		-- ('actorstatesverifreg'),
		-- ('blocks'),
		-- ('messages'),
		-- ('chaineconomics'),
		-- ('msapprovals'),
		-- ('implicitmessage'),
		-- ('consensus')
	) as x(task)
)
,

-- cross product of heights and tasks
all_heights_and_tasks_in_range as (
	select h.height, t.task
	from interesting_heights h
	cross join interesting_tasks t
)
,

-- all heights from processing reports which were
-- recorded (by gap_fill or consensus) that it is
-- a null round with no data to index.
-- then take cross product of these heights w tasks
null_round_heights_and_tasks_in_range as (
	select pr.height, t.task
	from visor_processing_reports pr
	cross join interesting_tasks t
	where pr.status_information = ?3
	and pr.height between ?0 and ?1
	group by 1, 2
)
,

-- all heights and tasks which need to be filled
all_incomplete_heights_and_tasks as (

	select height, task
		-- starting from the set of all heights and tasks
		-- in our range
    from all_heights_and_tasks_in_range

    -- remove all heights and tasks which have at least one OK
    except
    select height, task
    from visor_processing_reports
    where status = ?3
		and height between ?0 and ?1

    -- remove the null rounds by height and task
    except
    select height, task
    from null_round_heights_and_tasks_in_range
)

-- ordering for tidy persistence
select height, task
from all_incomplete_heights_and_tasks
order by 1 desc
;
`, strings.Join(sqlFmtTaskValues, ","))
	res, err := g.DB.AsORM().QueryContext(
		ctx,
		&result,
		query,
		g.minHeight,
		g.maxHeight,
		visor.ProcessingStatusInformationNullRound,
		visor.ProcessingStatusOK,
	)
	if err != nil {
		return nil, err
	}
	log.Debugw("executed find gap query and found epoch,task gaps", "count", res.RowsReturned())

	for _, r := range result {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		out = append(out, &visor.GapReport{
			Height:     int64(r.Height),
			Task:       r.Task,
			Status:     "GAP",
			Reporter:   g.name,
			ReportedAt: start,
		})
	}
	return out, nil
}
