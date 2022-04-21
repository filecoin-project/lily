package integrated

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/index/manager")

type Exporter interface {
	ExportResult(ctx context.Context, strg model.Storage, height int64, m []*indexer.ModelResult) error
}

// Manager manages the execution of an Indexer. It may be used to index TipSets both serially or in parallel.
type Manager struct {
	api          tasks.DataSource
	storage      model.Storage
	indexBuilder *Builder
	exporter     Exporter
	window       time.Duration
	name         string
}

type ManagerOpt func(i *Manager)

// WithWindow overrides the Manager's default (0) window with the provided window.
// The value of the window controls a timeout after which the Manager aborts processing the tipset, marking any incomplete
// tasks as SKIPPED.
func WithWindow(w time.Duration) ManagerOpt {
	return func(m *Manager) {
		m.window = w
	}
}

// NewManager returns a default Manager. Any provided ManagerOpt's will override Manager's default values.
func NewManager(api tasks.DataSource, strg model.Storage, name string, opts ...ManagerOpt) (*Manager, error) {
	im := &Manager{
		api:     api,
		storage: strg,
		window:  0,
		name:    name,
	}

	for _, opt := range opts {
		opt(im)
	}

	im.indexBuilder = NewBuilder(api, name)

	if im.exporter == nil {
		im.exporter = indexer.NewModelExporter(name)
	}
	return im, nil
}

// TipSet synchronously indexes and persists `ts`. TipSet returns an error if the Manager's Indexer encounters a
// fatal error. TipSet returns false if one or more of the Indexer's tasks complete with a status `ERROR` or `SKIPPED`, else returns true.
// Upon cancellation of `ctx` TipSet will persist all incomplete tasks with status `SKIPPED` before returning.
func (i *Manager) TipSet(ctx context.Context, ts *types.TipSet, priority string, tasks ...string) (bool, error) {
	ctx, span := otel.Tracer("").Start(ctx, "Manager.TipSet")
	defer span.End()
	lg := log.With("height", ts.Height(), "reporter", i.name)
	lg.Info("index tipset")

	var cancel func()
	var procCtx context.Context // cancellable context for the task
	if i.window > 0 {
		// Do as much indexing as possible in the specified time window (usually one epoch when following head of chain)
		// Anything not completed in that time will be marked as incomplete
		procCtx, cancel = context.WithTimeout(ctx, i.window)
	} else {
		// Ensure all goroutines are stopped when we exit
		procCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	idxer, err := i.indexBuilder.WithTasks(tasks).Build()
	if err != nil {
		return false, err
	}

	// asynchronously begin indexing tipset `ts`, returning results as they become avaiable.
	taskResults, taskErrors, err := idxer.TipSet(procCtx, ts)
	// indexer suffered fatal error, abort.
	if err != nil {
		return false, err
	}

	// there are no results, bail.
	if taskResults == nil {
		return true, nil
	}

	var modelResults []*indexer.ModelResult
	success := true
	// collect all the results, recording if any of the tasks were skipped or errored
	for res := range taskResults {
		select {
		case fatal := <-taskErrors:
			lg.Errorw("fatal indexer error", "error", fatal)
			return false, fatal
		default:
			for _, report := range res.Report {
				if report.Status != visormodel.ProcessingStatusOK &&
					report.Status != visormodel.ProcessingStatusInfo {
					lg.Warnw("task failed", "task", res.Name, "status", report.Status, "errors", report.ErrorsDetected, "info", report.StatusInformation)
					success = false
				} else {
					lg.Infow("task success", "task", res.Name, "status", report.Status, "duration", report.CompletedAt.Sub(report.StartedAt))
				}
			}
			modelResults = append(modelResults, &indexer.ModelResult{
				Name:  res.Name,
				Model: model.PersistableList{res.Report, res.Data},
			})

		}
	}

	// synchronously export extracted data and its report. If datas at this height are currently being persisted this method will block to avoid deadlocking the database.
	if err := i.exporter.ExportResult(ctx, i.storage, int64(ts.Height()), modelResults); err != nil {
		return false, err
	}

	lg.Infow("index tipset complete", "success", success)
	return success, nil
}
