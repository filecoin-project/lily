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
	log.With("type", "find")

	head, err := g.node.ChainHead(ctx)
	if err != nil {
		return err
	}
	if uint64(head.Height()) < g.maxHeight {
		return xerrors.Errorf("cannot look for gaps beyond chain head height %d", head.Height())
	}

	// looks for incomplete or missing epochs. An incomplete epoch has some, but
	// not all tasks in the processing report table. A missing epoch are heights
	// which do not exist at all in the processing report table.
	taskGaps, err := g.findMissingTasksAndEpochs(ctx)
	if err != nil {
		return xerrors.Errorf("finding task epoch gaps: %w", err)
	}

	return g.DB.PersistBatch(ctx, taskGaps)
}

type GapIndexerLens interface {
	ChainGetTipSetByHeight(ctx context.Context, epoch abi.ChainEpoch, tsk types.TipSetKey) (*types.TipSet, error)
}

type TaskHeight struct {
	Task   string
	Height uint64
	Status string
}

func (g *GapIndexer) findMissingTasksAndEpochs(ctx context.Context) (visor.GapReportList, error) {
	log.Debug("finding task epoch gaps for heights", g.minHeight, "through", g.maxHeight)
	start := time.Now()

	var (
		result           []TaskHeight
		out              visor.GapReportList
		sqlFmtTaskValues []string
	)

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
-- recorded (by gap_fill or consensus) and are
-- null rounds with no data to index.
-- take cross product of these heights w tasks
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
