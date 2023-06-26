package tipset

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/integrated/processor"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	taskapi "github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/integrated/tipset")

// TipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
type TipSetIndexer struct {
	name      string
	node      taskapi.DataSource
	taskNames []string

	processor *processor.StateProcessor
}

func (ti *TipSetIndexer) init() error {
	indexerTasks, err := tasktype.MakeTaskNames(ti.taskNames)
	if err != nil {
		return err
	}

	ti.processor, err = processor.New(ti.node, ti.name, indexerTasks)
	if err != nil {
		return err
	}

	return nil
}

type Result struct {
	// Name of the task executed.
	Name string
	// Data extracted during task execution.
	Data model.Persistable
	// Report containing details of task execution success and duration.
	Report visormodel.ProcessingReportList
}

// TipSet keeps no internal state and asynchronously indexes `ts` returning Result's as they extracted.
// If the TipSetIndexer encounters an error (fails to fetch ts's parent) it returns immediately and performs no work.
// If one of the TipSetIndexer's tasks encounters a fatal error, the error is return on the error channel.
func (ti *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) (chan *Result, chan error, error) {
	start := time.Now()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Name, ti.name))
	ctx, span := otel.Tracer("").Start(ctx, "TipSetIndexer.TipSet")

	if ts.Height() == 0 {
		// bail, the parent of genesis is itself, there is no diff
		return nil, nil, nil
	}

	var executed, current *types.TipSet
	pts, err := ti.node.TipSet(ctx, ts.Parents())
	if err != nil {
		return nil, nil, err
	}
	current = ts
	executed = pts

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current_tipset", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed_tipset", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
			attribute.String("name", ti.name),
			attribute.StringSlice("tasks", ti.taskNames),
		)
	}

	log.Infow("index", "reporter", ti.name, "current", current.Height(), "executed", executed.Height())
	stateResults, taskNames := ti.processor.State(ctx, current, executed)

	// build list of executing tasks, used below to label incomplete tasks as skipped.
	executingTasks := make(map[string]bool, len(taskNames))
	for _, name := range taskNames {
		executingTasks[name] = false
	}

	var (
		outCh = make(chan *Result, len(stateResults))
		errCh = make(chan error, len(stateResults))
	)
	go func() {
		defer func() {
			close(outCh)
			close(errCh)
			defer span.End()
		}()

		for res := range stateResults {
			select {
			// canceled, we ran out of time. Mark incomplete work as skipped and exit.
			case <-ctx.Done():
				for name, complete := range executingTasks {
					if complete {
						continue
					}
					stats.Record(ctx, metrics.TipSetSkip.M(1))
					outCh <- &Result{
						Name: name,
						Data: nil,
						Report: visormodel.ProcessingReportList{
							&visormodel.ProcessingReport{
								Height:            int64(current.Height()),
								StateRoot:         current.ParentState().String(),
								Reporter:          ti.name,
								Task:              name,
								StartedAt:         start,
								CompletedAt:       time.Now(),
								Status:            visormodel.ProcessingStatusSkip,
								StatusInformation: "indexer timeout",
							}},
					}
				}
				return
			default:
				// received a result

				llt := log.With("height", current.Height(), "task", res.Task, "reporter", ti.name)

				// Was there a fatal error?
				if res.Error != nil {
					llt.Errorw("task returned with error", "error", res.Error.Error())
					errCh <- res.Error
				}
				// processor is complete if we receive a result
				executingTasks[res.Task] = true

				if res.Report == nil || len(res.Report) == 0 {
					// Nothing was done for this tipset
					llt.Debugw("task returned with no report")
					continue
				}

				for idx := range res.Report {
					// Fill in some report metadata
					res.Report[idx].Reporter = ti.name
					res.Report[idx].Task = res.Task
					res.Report[idx].StartedAt = res.StartedAt
					res.Report[idx].CompletedAt = res.CompletedAt

					if err := res.Report[idx].ErrorsDetected; err != nil {
						// because error is just an interface it may hold a value of any concrete type that implements it, and if
						// said type has unexported fields json marshaling will fail when persisting.
						e, ok := err.(error)
						if ok {
							res.Report[idx].ErrorsDetected = &struct {
								Error string
							}{Error: e.Error()}
						}
						res.Report[idx].Status = visormodel.ProcessingStatusError
					} else if res.Report[idx].StatusInformation != "" {
						res.Report[idx].Status = visormodel.ProcessingStatusInfo
					} else {
						res.Report[idx].Status = visormodel.ProcessingStatusOK
					}

					llt.Debugw("task report", "status", res.Report[idx].Status, "duration", res.Report[idx].CompletedAt.Sub(res.Report[idx].StartedAt))
				}

				// Persist the processing report and the data in a single transaction
				outCh <- &Result{
					Name:   res.Task,
					Data:   res.Data,
					Report: res.Report,
				}
			}
		}
	}()

	return outCh, errCh, nil
}
