package gap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/testutil"
)

var (
	minHeight = uint64(0)
	maxHeight = uint64(10)
)

func TestFind(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	// TODO adjust timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	t.Run("gap all tasks at epoch 1", func(t *testing.T) {
		truncate(t, db)
		gapHeight := int64(1)
		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.deleteEpochStatus(gapHeight, visor.ProcessingStatusOK)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(gapHeight, tasktype.AllTableTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("gap all tasks at epoch 1 4 5", func(t *testing.T) {
		truncate(t, db)
		gapHeights := []int64{1, 4, 5}
		gapTasks := tasktype.AllTableTasks

		pre := NewPREditor(t, db, t.Name())
		pre.truncate()
		pre.initialize(maxHeight, tasktype.AllTableTasks...)

		var expected visor.GapReportList
		for _, height := range gapHeights {
			pre.deleteEpochStatus(height, visor.ProcessingStatusOK)
			expected = append(expected, makeGapReportList(height, gapTasks...)...)
		}

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, gapTasks).Find(ctx)
		require.NoError(t, err)

		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("gap at epoch 2 for miner and init task", func(t *testing.T) {
		truncate(t, db)
		gapHeight := int64(2)
		gapTasks := []string{tasktype.MinerInfo, tasktype.IdAddress}

		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.deleteEpochStatus(gapHeight, visor.ProcessingStatusOK, WithTasks(gapTasks...))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(gapHeight, gapTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("gap at epoch 2 for miner and init task epoch 10 blocks messages market", func(t *testing.T) {
		truncate(t, db)
		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(tasktype.MinerInfo, tasktype.IdAddress))
		pre.deleteEpochStatus(10, visor.ProcessingStatusOK, WithTasks(tasktype.BlockHeader, tasktype.Message, tasktype.MarketDealProposal))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, tasktype.MinerInfo, tasktype.IdAddress)
		expected = append(expected, makeGapReportList(10, tasktype.BlockHeader, tasktype.Message, tasktype.MarketDealProposal)...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("skip all tasks at epoch 1 and miner task at epoch 5", func(t *testing.T) {
		truncate(t, db)
		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.updateEpochStatus(1, visor.ProcessingStatusSkip)
		pre.updateEpochStatus(5, visor.ProcessingStatusSkip, WithTasks(tasktype.MinerInfo))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(1, tasktype.AllTableTasks...)
		expected = append(expected, makeGapReportList(5, tasktype.MinerInfo)...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("gap at epoch 2 for miner and init task, miner errors in 8, all errors in 9", func(t *testing.T) {
		truncate(t, db)
		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)

		pre.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(tasktype.MinerInfo, tasktype.IdAddress))
		pre.updateEpochStatus(8, visor.ProcessingStatusError, WithTasks(tasktype.MinerInfo))
		pre.updateEpochStatus(9, visor.ProcessingStatusError)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, tasktype.MinerInfo, tasktype.IdAddress)
		expected = append(expected, makeGapReportList(8, tasktype.MinerInfo)...)
		expected = append(expected, makeGapReportList(9, tasktype.AllTableTasks...)...)
		assertGapReportsEqual(t, expected, actual)
	})

	// ensure that when there is more than one processing entry for a height we handle is correctly
	t.Run("duplicate processing row with gap at epoch 2 for miner and init task", func(t *testing.T) {
		truncate(t, db)
		pre1 := NewPREditor(t, db, "reporter1")
		pre2 := NewPREditor(t, db, "reporter2")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre2.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(tasktype.MinerInfo, tasktype.IdAddress))
		pre2.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(tasktype.MinerInfo, tasktype.IdAddress))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, tasktype.MinerInfo, tasktype.IdAddress)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("(sub task indexer, full reports table) gap at epoch 2 for messages and init task", func(t *testing.T) {
		truncate(t, db)

		gapTasks := []string{tasktype.Message, tasktype.IdAddress}
		monitoringTasks := append(gapTasks, []string{tasktype.BlockHeader, tasktype.ChainEconomics}...)

		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(gapTasks...))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		// tasks to find gaps in
		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, monitoringTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, gapTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("(sub task indexer partial reports table) gap at epoch 2 for messages and init task", func(t *testing.T) {
		truncate(t, db)

		// tasks to create gaps for
		gapTasks := []string{tasktype.Message, tasktype.IdAddress}
		monitoringTasks := append(gapTasks, []string{tasktype.BlockHeader, tasktype.ChainEconomics}...)

		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, monitoringTasks...)
		pre.deleteEpochStatus(2, visor.ProcessingStatusOK, WithTasks(gapTasks...))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, monitoringTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, gapTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("(#775) for each task at epoch 2 there exists an ERROR", func(t *testing.T) {
		truncate(t, db)

		pre := NewPREditor(t, db, t.Name())
		pre.initialize(maxHeight, tasktype.AllTableTasks...)
		pre.updateEpochStatus(2, visor.ProcessingStatusError, WithTasks(tasktype.AllTableTasks...))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, tasktype.AllTableTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("(#775) for init and miner tasks at epoch 2 there exists an ERROR _and_ an OK", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre2 := NewPREditor(t, db, "reporter2")
		pre2.insertEpochStatus(2, visor.ProcessingStatusError, WithTasks(tasktype.IdAddress, tasktype.MinerInfo))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		// expect nothing since tasks have an OK status dispite the error
		require.Len(t, actual, 0)
	})

	t.Run("(#773) for each task at epoch 2 there exists a SKIP and an OK", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.updateEpochStatus(2, visor.ProcessingStatusSkip)

		pre2 := NewPREditor(t, db, "reporter2")
		pre2.insertEpochStatus(2, visor.ProcessingStatusOK)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		// no gaps should be found since the epoch has OK's for all tasks; the SKIPS are ignored.
		require.Len(t, actual, 0)
	})

	t.Run("for each task at epoch 2 and 8 there exists a SKIP, ERROR and an OK", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.updateEpochStatus(2, visor.ProcessingStatusSkip)
		pre1.updateEpochStatus(8, visor.ProcessingStatusSkip)

		pre2 := NewPREditor(t, db, "reporter2")
		pre2.insertEpochStatus(2, visor.ProcessingStatusError)
		pre2.insertEpochStatus(8, visor.ProcessingStatusError)

		pre3 := NewPREditor(t, db, "reporter3")
		pre3.insertEpochStatus(2, visor.ProcessingStatusOK)
		pre3.insertEpochStatus(8, visor.ProcessingStatusOK)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		// no gaps should be found since the epoch has OK's for all tasks; the SKIPS and ERRORs are ignored.
		require.Len(t, actual, 0)
	})

	t.Run("for each task at epoch 2 and 8 there exists a SKIP, ERROR and duplicate OKs", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.updateEpochStatus(2, visor.ProcessingStatusSkip)
		pre1.updateEpochStatus(8, visor.ProcessingStatusSkip)

		pre2 := NewPREditor(t, db, "reporter2")
		pre2.initialize(maxHeight, tasktype.AllTableTasks...)
		pre2.updateEpochStatus(2, visor.ProcessingStatusError)
		pre2.updateEpochStatus(8, visor.ProcessingStatusError)

		pre3 := NewPREditor(t, db, "reporter3")
		pre3.initialize(maxHeight, tasktype.AllTableTasks...)
		pre3.updateEpochStatus(2, visor.ProcessingStatusOK)
		pre3.updateEpochStatus(8, visor.ProcessingStatusOK)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		// no gaps should be found since the epoch has OK's for all tasks; the SKIPS and ERRORs are ignored.
		require.Len(t, actual, 0)
	})

	t.Run("for each task at epoch 2 there exists a SKIP and ERROR", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.updateEpochStatus(2, visor.ProcessingStatusSkip)

		pre2 := NewPREditor(t, db, "reporter2")
		pre2.updateEpochStatus(2, visor.ProcessingStatusError)

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(2, tasktype.AllTableTasks...)
		assertGapReportsEqual(t, expected, actual)
	})

	t.Run("null rounds at epoch 2 and non-null round tasks at epoch 3", func(t *testing.T) {
		truncate(t, db)

		pre1 := NewPREditor(t, db, "reporter1")
		pre1.initialize(maxHeight, tasktype.AllTableTasks...)
		pre1.updateEpochStatus(2, visor.ProcessingStatusInfo, WithStatusInformation(visor.ProcessingStatusInformationNullRound))
		pre1.updateEpochStatus(3, visor.ProcessingStatusInfo, WithStatusInformation("not the permitted null round"))

		strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
		require.NoError(t, err, "NewDatabaseFromDB")

		actual, err := NewFinder(nil, strg, t.Name(), minHeight, maxHeight, tasktype.AllTableTasks).Find(ctx)
		require.NoError(t, err)

		expected := makeGapReportList(3, tasktype.AllTableTasks...)
		assertGapReportsEqual(t, expected, actual)
	})
}

type assertFields struct {
	status string
	task   string
}

func assertGapReportsEqual(t testing.TB, expected, actual visor.GapReportList) {
	assert.Equal(t, len(expected), len(actual))
	exp := make(map[int64][]assertFields, len(expected))
	act := make(map[int64][]assertFields, len(expected))

	for _, e := range expected {
		exp[e.Height] = append(exp[e.Height], assertFields{
			status: e.Status,
			task:   e.Task,
		})
	}

	for _, a := range actual {
		act[a.Height] = append(act[a.Height], assertFields{
			status: a.Status,
			task:   a.Task,
		})
	}

	for epoch := range exp {
		e := exp[epoch]
		a := act[epoch]
		sort.Slice(e, func(i, j int) bool {
			return e[i].task < e[j].task
		})
		sort.Slice(a, func(i, j int) bool {
			return a[i].task < a[j].task
		})
		assert.Equal(t, e, a)
	}
}

func makeGapReportList(height int64, tasks ...string) visor.GapReportList {
	var out visor.GapReportList
	for _, task := range tasks {
		out = append(out, &visor.GapReport{
			Height:     height,
			Task:       task,
			Status:     "GAP",
			Reporter:   "gapIndexer",
			ReportedAt: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
		})
	}
	return out
}

func truncate(tb testing.TB, db *pg.DB) {
	_, err := db.Exec(`TRUNCATE TABLE visor_processing_reports`)
	require.NoError(tb, err, "visor_processing_report")
}

type PREditor struct {
	t        testing.TB
	db       *pg.DB
	reporter string
}

func NewPREditor(tb testing.TB, db *pg.DB, reporter string) *PREditor {
	return &PREditor{
		t:        tb,
		db:       db,
		reporter: reporter,
	}
}

type PREditorQuery struct {
	epoch             int64
	tasks             []string
	status            string
	statusInformation string
}

type PREditorOption func(q *PREditorQuery)

func WithTasks(tasks ...string) PREditorOption {
	return func(q *PREditorQuery) {
		q.tasks = tasks
	}
}

func WithStatusInformation(statusInformation string) PREditorOption {
	return func(q *PREditorQuery) {
		q.statusInformation = statusInformation
	}
}

func (e *PREditor) truncate() {
	_, err := e.db.Exec(`TRUNCATE TABLE visor_processing_reports`)
	require.NoError(e.t, err, "visor_processing_report")
}

func (e *PREditor) initialize(count uint64, tasks ...string) {
	// build the task array
	// uncomment to see all query
	// db.AddQueryHook(&LoggingQueryHook{})
	taskQbuilder := strings.Builder{}
	for idx, t := range tasks {
		taskQbuilder.WriteString("'")
		taskQbuilder.WriteString(t)
		taskQbuilder.WriteString("'")
		if idx != len(tasks)-1 {
			taskQbuilder.WriteString(",")
		}
	}
	query := fmt.Sprintf(`do $$
    DECLARE
        -- TODO add internal messages
        task_name text;
        arr text[] := array[%s];
    begin
        for epoch in 0..%d loop
                foreach task_name in array arr loop
                insert into public.visor_processing_reports(height, state_root, reporter, task, started_at, completed_at, status, status_information, errors_detected)
                values(epoch, concat(epoch, '_state_root'), '%s', task_name, '2021-01-01 00:00:00.000000 +00:00', '2021-01-21 00:00:00.000000 +00:00', 'OK', null, null);
                    end loop;
            end loop;
    end;
$$;`, taskQbuilder.String(), count, e.reporter)
	_, err := e.db.Exec(query)

	require.NoError(e.t, err)
}

func (e *PREditor) updateEpochStatus(epoch int64, status string, opts ...PREditorOption) {
	q := &PREditorQuery{
		epoch:  epoch,
		status: status,
		tasks:  tasktype.AllTableTasks,
	}
	for _, opt := range opts {
		opt(q)
	}
	for _, task := range q.tasks {
		_, err := e.db.Exec(
			`
	update visor_processing_reports
	set status = ?, status_information = ?
	where height = ? and task = ? and reporter = ?
`,
			q.status, q.statusInformation, q.epoch, task, e.reporter)
		require.NoError(e.t, err)
	}
}

func (e *PREditor) insertEpochStatus(epoch int64, status string, opts ...PREditorOption) {
	q := &PREditorQuery{
		epoch:  epoch,
		status: status,
		tasks:  tasktype.AllTableTasks,
	}
	for _, opt := range opts {
		opt(q)
	}
	for _, task := range q.tasks {
		qsrt := fmt.Sprintf(`
	insert into public.visor_processing_reports(height, state_root, reporter, task, started_at, completed_at, status, status_information, errors_detected)
	values(%d, concat(%d, '_state_root'), '%s', '%s', '2021-01-01 00:00:00.000000 +00:00', '2021-01-21 00:00:00.000000 +00:00', '%s', null, null);
			`, q.epoch, q.epoch, e.reporter, task, q.status)
		_, err := e.db.Exec(qsrt)
		require.NoError(e.t, err)
	}
}

func (e *PREditor) deleteEpochStatus(epoch int64, status string, opts ...PREditorOption) {
	q := &PREditorQuery{
		epoch:  epoch,
		status: status,
		tasks:  tasktype.AllTableTasks,
	}
	for _, opt := range opts {
		opt(q)
	}
	for _, task := range q.tasks {
		_, err := e.db.Exec(
			`
	delete from visor_processing_reports
	where height = ? and task = ? and status = ? and reporter = ?
`,
			q.epoch, task, q.status, e.reporter)
		require.NoError(e.t, err)
	}
}

type LoggingQueryHook struct{}

func (l *LoggingQueryHook) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	q, err := event.FormattedQuery()
	if err != nil {
		return nil, err
	}

	if event.Err != nil {
		fmt.Printf("%s executing a query:\n%s\n", event.Err, q)
	}
	fmt.Println(string(q))

	return ctx, nil
}

func (l *LoggingQueryHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	return nil
}
