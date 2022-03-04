package integrated

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/chain/index"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/index/manager")

type Indexer interface {
	TipSet(ctx context.Context, ts *types.TipSet) (chan *Result, chan error, error)
}

type Exporter interface {
	ExportResult(ctx context.Context, strg model.Storage, m *index.ModelResult) error
}

// Manager manages the execution of an Indexer. It may be used to index TipSets both serially or in parallel.
type Manager struct {
	api      tasks.DataSource
	storage  model.Storage
	indexer  Indexer
	exporter Exporter
	window   time.Duration

	// used for async tipset indexing
	pool   *workerpool.WorkerPool
	active int64 // must be accessed using atomic operations, updated automatically.

	fatalMu sync.Mutex
	fatal   error
}

type ManagerOpt func(i *Manager)

// WithWorkerPool overrides the Manager's default pool (1) with the provided pool.
// The size of the pool controls the number of TipSets that can be indexed in parallel.
func WithWorkerPool(pool *workerpool.WorkerPool) ManagerOpt {
	return func(m *Manager) {
		m.pool = pool
	}
}

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
		im.exporter = index.NewModelExporter()
	}

	if im.pool == nil {
		im.pool = workerpool.New(1)
	}
	return im, nil
}

// TipSetAsync enqueues `ts` into the Manager's worker pool for processing. An error is returned if any Indexer in
// the pool encounters a fatal error, else the method returns immediately.
func (i *Manager) TipSetAsync(ctx context.Context, ts *types.TipSet) error {
	if err := i.fatalError(); err != nil {
		defer i.pool.Stop()
		return err
	}

	stats.Record(ctx, metrics.IndexManagerActiveWorkers.M(i.active))
	stats.Record(ctx, metrics.IndexManagerWaitingWorkers.M(int64(i.pool.WaitingQueueSize())))
	if i.pool.WaitingQueueSize() > i.pool.Size() {
		log.Warnw("queuing worker in Manager pool", "waiting", i.pool.WaitingQueueSize())
	}
	log.Infow("submitting tipset for async indexing", "height", ts.Height(), "active", i.active)

	ctx, span := otel.Tracer("").Start(ctx, "Manager.TipSetAsync")
	i.pool.Submit(func() {
		atomic.AddInt64(&i.active, 1)
		defer func() {
			atomic.AddInt64(&i.active, -1)
			span.End()
		}()

		ts := ts
		success, err := i.TipSet(ctx, ts)
		if err != nil {
			log.Errorw("index manager suffered fatal error", "error", err, "height", ts.Height(), "tipset", ts.Key().String())
			i.setFatalError(err)
			return
		}
		if !success {
			log.Warnw("index manager failed to fully index tipset", "height", ts.Height(), "tipset", ts.Key().String())
		}
	})
	return nil
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

	success := true
	for {
		select {
		case fatal := <-taskErrors:
			log.Errorw("fatal indexer error", "error", fatal)
			return false, fatal
		default:
			res, ok := <-taskResults
			if !ok {
				log.Infow("index tipset complete", "height", ts.Height(), "success", success)
				return success, nil
			}

			for _, report := range res.Report {
				if report.Status != visormodel.ProcessingStatusOK &&
					report.Status != visormodel.ProcessingStatusInfo {
					log.Infow("task failed", "task", res.Name, "status", report.Status, "errors", report.ErrorsDetected, "info", report.StatusInformation)
					success = false
				} else {
					log.Infow("task success", "task", res.Name, "status", report.Status, "duration", report.CompletedAt.Sub(report.StartedAt))
				}
			}
			m := &index.ModelResult{
				Name:  res.Name,
				Model: model.PersistableList{res.Report, res.Data},
			}

			// synchronously export extracted data and its report.
			if err := i.exporter.ExportResult(ctx, i.storage, m); err != nil {
				return false, err
			}
		}
	}
}

func (i *Manager) setFatalError(err error) {
	i.fatalMu.Lock()
	i.fatal = err
	i.fatalMu.Unlock()
}

func (i *Manager) fatalError() error {
	i.fatalMu.Lock()
	out := i.fatal
	i.fatalMu.Unlock()
	return out
}
