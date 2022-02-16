package chain

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opencensus.io/stats"

	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
)

var _ TipSetObserver = (*IndexManager)(nil)

type IndexManager struct {
	name     string
	api      task.TaskAPI
	storage  model.Storage
	indexer  *TipSetIndexer
	exporter *ModelExporter
	window   time.Duration
}

func NewIndexManager(api task.TaskAPI, strg model.Storage, name string, tasks []string, exporter *ModelExporter, window time.Duration) (*IndexManager, error) {
	indexer, err := NewTipSetIndexer(api, name, tasks)
	if err != nil {
		return nil, err
	}
	return &IndexManager{
		name:     "index-manager",
		api:      api,
		storage:  strg,
		indexer:  indexer,
		exporter: exporter,
		window:   window,
	}, nil
}

func (i *IndexManager) TipSet(ctx context.Context, ts *types.TipSet) (bool, error) {
	if !i.indexer.Ready() {
		return false, i.SkipUnprocessedTipSets(ctx, ts)
	}

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

	taskResults, taskErrors, err := i.indexer.TipSet(procCtx, ts)
	// indexer suffered fatal error
	if err != nil {
		return false, err
	}

	// there are no results
	if taskResults == nil {
		return true, nil
	}

	success := true
	for {
		select {
		case <-procCtx.Done():
			return success, i.SkipUnprocessedTipSets(ctx, ts)
		case fatal := <-taskErrors:
			return false, fatal
		default:
			res, ok := <-taskResults
			if !ok {
				return success, nil
			}

			for _, report := range res.Report {
				if report.Status != visormodel.ProcessingStatusOK &&
					report.Status != visormodel.ProcessingStatusInfo {

					success = false
				}
			}
			m := &ModelResult{
				Name:  res.Name,
				Model: model.PersistableList{res.Report, res.Data},
			}

			if err := i.exporter.ExportResult(ctx, i.storage, m); err != nil {
				return false, err
			}
		}
	}
}

func (i *IndexManager) Ready() bool {
	return i.indexer.Ready()
}

func (i *IndexManager) Close() error {
	return nil
}

func (i *IndexManager) SkipUnprocessedTipSets(ctx context.Context, ts *types.TipSet) error {
	skipTime := time.Now()
	for _, name := range i.indexer.processor.IncompleteTasks() {
		log.Warnw("task skipped", "task", name, "reason", "state processor deadline exceeded")
		if err := i.exporter.ExportResult(ctx, i.storage, &ModelResult{
			Name:  name,
			Model: buildSkippedReport(ts, i.name, name, skipTime, "deadline exceeded"),
		}); err != nil {
			return err
		}
	}
	stats.Record(ctx, metrics.TipSetSkip.M(1))
	return nil
}

func buildSkippedReport(ts *types.TipSet, reporter, taskName string, timestamp time.Time, reason string) *visormodel.ProcessingReport {
	return &visormodel.ProcessingReport{
		Height:            int64(ts.Height()),
		StateRoot:         ts.ParentState().String(),
		Reporter:          reporter,
		Task:              taskName,
		StartedAt:         timestamp,
		CompletedAt:       timestamp,
		Status:            visormodel.ProcessingStatusSkip,
		StatusInformation: reason,
	}
}
