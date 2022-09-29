package integrated

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/integrated/tipset"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var log = logging.Logger("lily/index/manager")

type Exporter interface {
	ExportResult(ctx context.Context, strg model.Storage, height int64, m []*indexer.ModelResult) error
}

// Manager manages the execution of an Indexer. It may be used to index TipSets both serially or in parallel.
type Manager struct {
	storage      model.Storage
	indexBuilder tipset.IndexerBuilder
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

func WithExporter(e Exporter) ManagerOpt {
	return func(m *Manager) {
		m.exporter = e
	}
}

// NewManager returns a default Manager. Any provided ManagerOpt's will override Manager's default values.
func NewManager(strg model.Storage, idxBuilder tipset.IndexerBuilder, opts ...ManagerOpt) (*Manager, error) {
	im := &Manager{
		storage: strg,
		window:  0,
		name:    idxBuilder.Name(),
	}

	for _, opt := range opts {
		opt(im)
	}

	im.indexBuilder = idxBuilder

	if im.exporter == nil {
		im.exporter = indexer.NewModelExporter(idxBuilder.Name())
	}
	return im, nil
}

// TipSet synchronously indexes and persists `ts`. TipSet returns an error if the Manager's Indexer encounters a
// fatal error. TipSet returns false if one or more of the Indexer's tasks complete with a status `ERROR` or `SKIPPED`, else returns true.
// Upon cancellation of `ctx` TipSet will persist all incomplete tasks with status `SKIPPED` before returning.
func (i *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (bool, error) {
	opts, err := indexer.ConstructOptions(options...)
	if err != nil {
		return false, err
	}
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

	idxer, err := i.indexBuilder.WithTasks(opts.Tasks).Build()
	if err != nil {
		return false, err
	}

	// asynchronously begin indexing tipset `ts`, returning results as they become available.
	taskResults, taskErrors, err := idxer.TipSet(procCtx, ts)
	// indexer suffered fatal error, abort.
	if err != nil {
		return false, err
	}

	// there are no results, bail.
	if taskResults == nil {
		return true, nil
	}

	success := atomic.NewBool(true)
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		var modelResults []*indexer.ModelResult
		// collect all the results, recording if any of the tasks were skipped or errored
		for res := range taskResults {
			for _, report := range res.Report {
				if report.Status != visormodel.ProcessingStatusOK &&
					report.Status != visormodel.ProcessingStatusInfo {
					lg.Warnw("task failed", "task", res.Name, "status", report.Status, "errors", report.ErrorsDetected, "info", report.StatusInformation)
					success.Store(false)
				} else {
					lg.Infow("task success", "task", res.Name, "status", report.Status, "duration", report.CompletedAt.Sub(report.StartedAt))
				}
			}
			modelResults = append(modelResults, &indexer.ModelResult{
				Name:  res.Name,
				Model: model.PersistableList{res.Report, res.Data},
			})

		}

		// synchronously export extracted data and its report. If datas at this height are currently being persisted this method will block to avoid deadlocking the database.
		if err := i.exporter.ExportResult(ctx, i.storage, int64(ts.Height()), modelResults); err != nil {
			return err
		}

		return nil
	})

	grp.Go(func() error {
		for fatal := range taskErrors {
			success.Store(false)
			return fatal
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		return false, err
	}

	return success.Load(), nil
}
