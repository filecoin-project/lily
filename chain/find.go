package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/go-state-types/abi"
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
	head, err := g.node.ChainHead(ctx)
	if err != nil {
		return err
	}
	if uint64(head.Height()) < g.maxHeight {
		return xerrors.Errorf("cannot look for gaps beyond chain head height %d", head.Height())
	}

	findLog := log.With("type", "find")

	// looks for tasks within epochs which haven't sucessfully completed (at least 1 OK or NULL_ROUND)
	taskGaps, epochNulls, err := g.findMissingEpochsAndTasks(ctx, g.node)
	if err != nil {
		return xerrors.Errorf("finding task epoch gaps: %w", err)
	}
	findLog.Infow("found gaps in tasks", "count", len(taskGaps))
	return g.DB.PersistBatch(ctx, taskGaps, epochNulls)
}

type taskHeight struct {
	Task   string
	Height uint64
	Status string
}

type GapIndexerLens interface {
	ChainGetTipSetByHeight(ctx context.Context, epoch abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error)
}

func (g *GapIndexer) findMissingEpochsAndTasks(ctx context.Context, node GapIndexerLens) (visor.GapReportList, visor.ProcessingReportList, error) {
	log.Debug("finding task epoch gaps")
	start := time.Now()
	var result []taskHeight

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
		-- example string (provided by sqlFmtTasks):
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

	-- starting from the set of all heights and tasks
	-- in our range
	select height, task
	from all_heights_and_tasks_in_range

		except

	-- remove all heights and tasks which have at least one OK
	select height, task
	from visor_processing_reports
	where status = ?3
	and height between ?0 and ?1

		except

	-- remove the null rounds by height and task
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
		return nil, nil, err
	}
	log.Debugw("executed find gap query and found epoch,task gaps", "count", res.RowsReturned())

	nullEpochs := make([]*visor.ProcessingReport, 0)
	gapRounds := make(visor.GapReportList, 0)
	cachedTipsetLookups := make(map[uint64]*types.TipSet)
	for _, r := range result {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		gh := abi.ChainEpoch(r.Height)

		// let's not call the API more than necessary
		var resultTS *types.TipSet
		if cachedTS, ok := cachedTipsetLookups[r.Height]; ok {
			resultTS = cachedTS
		} else {
			tsgap, err := node.ChainGetTipSetByHeight(ctx, gh, types.EmptyTSK)
			if err != nil {
				return nil, nil, err
			}
			cachedTipsetLookups[r.Height] = tsgap
			resultTS = tsgap
		}

		if resultTS.Height() == gh {
			// gap is just a gap
			gapRounds = append(gapRounds, g.buildGapReport(r.Height, r.Task, start))
		} else {
			// gap is a null round
			log.Debugw("found null round", "height", gh)
			nullEpochs = append(nullEpochs, g.buildNullReport(r.Height, start))
		}
	}
	return gapRounds, nullEpochs, nil
}

func (g *GapIndexer) buildNullReport(height uint64, start time.Time) *visor.ProcessingReport {
	return &visor.ProcessingReport{
		Height:            int64(height),
		StateRoot:         "NULL_ROUND", // let gap fill add the correct state root for this epoch when it runs the consensus task.
		Reporter:          g.name,
		Task:              "gap_find",
		StartedAt:         start,
		CompletedAt:       time.Now(),
		Status:            visor.ProcessingStatusInfo,
		StatusInformation: visor.ProcessingStatusInformationNullRound, // the gap finding logic uses this value to exclude null rounds from gap report.
	}
}

func (g *GapIndexer) buildGapReport(height uint64, task string, start time.Time) *visor.GapReport {
	return &visor.GapReport{
		Height:     int64(height),
		Task:       task,
		Status:     "GAP",
		Reporter:   g.name,
		ReportedAt: start,
	}
}
