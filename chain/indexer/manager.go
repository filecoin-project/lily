package indexer

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/index/manager")

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet) (chan *Result, chan error, error)
}

type Exporter interface {
	ExportResult(ctx context.Context, strg model.Storage, height int64, m []*ModelResult) error
}

// Manager manages the execution of an Indexer. It may be used to index TipSets both serially or in parallel.
type Manager struct {
	api      tasks.DataSource
	storage  model.Storage
	indexer  Indexer
	exporter Exporter
	window   time.Duration
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

// WithExporter overrides the Manager's default Exporter with the provided Exporter.
// An Exporter is used to export the results of the Manager's Indexer.
func WithExporter(e Exporter) ManagerOpt {
	return func(m *Manager) {
		m.exporter = e
	}
}

// WithIndexer overrides the Manager's default Indexer with the provided Indexer.
// An Indexer is used to collect state from a tipset.
func WithIndexer(i Indexer) ManagerOpt {
	return func(m *Manager) {
		m.indexer = i
	}
}

// NewManager returns a default Manager. Any provided ManagerOpt's will override Manager's default values.
func NewManager(api tasks.DataSource, strg model.Storage, name string, tasks []string, opts ...ManagerOpt) (*Manager, error) {
	im := &Manager{
		api:     api,
		storage: strg,
		window:  0,
	}

	for _, opt := range opts {
		opt(im)
	}

	if im.indexer == nil {
		var err error
		im.indexer, err = NewTipSetIndexer(api, name, tasks)
		if err != nil {
			return nil, err
		}
	}

	if im.exporter == nil {
		im.exporter = NewModelExporter()
	}
	return im, nil
}

// TipSet synchronously indexes and persists `ts`. TipSet returns an error if the Manager's Indexer encounters a
// fatal error. TipSet returns false if one or more of the Indexer's tasks complete with a status `ERROR` or `SKIPPED`, else returns true.
// Upon cancellation of `ctx` TipSet will persist all incomplete tasks with status `SKIPPED` before returning.
func (i *Manager) TipSet(ctx context.Context, ts *types.TipSet) (bool, error) {
	ctx, span := otel.Tracer("").Start(ctx, "Manager.TipSet")
	defer span.End()
	log.Infow("index tipset", "height", ts.Height())

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

	// asynchronously begin indexing tipset `ts`, returning results as they become avaiable.
	taskResults, taskErrors, err := i.indexer.TipSet(procCtx, ts)
	// indexer suffered fatal error, abort.
	if err != nil {
		return false, err
	}

	// there are no results, bail.
	if taskResults == nil {
		return true, nil
	}

	var modelResults []*ModelResult
	success := true
	// collect all the results, recording if any of the tasks were skipped or errored
	for res := range taskResults {
		select {
		case fatal := <-taskErrors:
			log.Errorw("fatal indexer error", "height", ts.Height(), "error", fatal)
			return false, fatal
		default:
			for _, report := range res.Report {
				if report.Status != visormodel.ProcessingStatusOK &&
					report.Status != visormodel.ProcessingStatusInfo {
					log.Warnw("task failed", "height", ts.Height(), "task", res.Name, "status", report.Status, "errors", report.ErrorsDetected, "info", report.StatusInformation)
					success = false
				} else {
					log.Infow("task success", "height", ts.Height(), "task", res.Name, "status", report.Status, "duration", report.CompletedAt.Sub(report.StartedAt))
				}
			}
			modelResults = append(modelResults, &ModelResult{
				Name:  res.Name,
				Model: model.PersistableList{res.Report, res.Data},
			})

		}
	}

	// synchronously export extracted data and its report. If datas at this height are currently being persisted this method will block to avoid deadlocking the database.
	if err := i.exporter.ExportResult(ctx, i.storage, int64(ts.Height()), modelResults); err != nil {
		return false, err
	}

	log.Infow("index tipset complete", "height", ts.Height(), "success", success)
	return success, nil
}
