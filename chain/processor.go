package chain

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opencensus.io/stats"

	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/indexer"
)

func NewStateProcessorBuilder(api task.TaskAPI, name string) *StateProcessorBuilder {
	return &StateProcessorBuilder{api: api, name: name}
}

type StateProcessorBuilder struct {
	options []func(sp *StateProcessor)
	api     task.TaskAPI
	name    string
}

func (spb *StateProcessorBuilder) add(cb func(sp *StateProcessor)) {
	spb.options = append(spb.options, cb)
}

func (spb *StateProcessorBuilder) WithTipSetProcessors(opt map[string]TipSetProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) WithTipSetsProcessors(opt map[string]TipSetsProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.tipsetsProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) WithActorProcessors(opt map[string]ActorProcessor) *StateProcessorBuilder {
	spb.add(func(sp *StateProcessor) {
		sp.actorProcessors = opt
	})
	return spb
}

func (spb *StateProcessorBuilder) Build() *StateProcessor {
	api := spb.api

	sp := &StateProcessor{
		name: spb.name,
		builtinProcessors: map[string]BuiltinProcessor{
			"builtin": indexer.NewTask(api),
		},
		api: spb.api,
	}

	for _, opt := range spb.options {
		opt(sp)
	}

	// build the taskName list
	for name := range sp.builtinProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.tipsetProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.tipsetsProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	for name := range sp.actorProcessors {
		sp.taskNames = append(sp.taskNames, name)
	}
	return sp
}

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type TipSetsProcessor interface {
	// ProcessTipSets processes sequential tipsts (a parent and a child, or an executed and a current). If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessActors(ctx context.Context, current *types.TipSet, executed *types.TipSet, actors task.ActorStateChangeDiff) (model.Persistable, *visormodel.ProcessingReport, error)
}

// Other names could be: SystemProcessor, ReportProcessor, IndexProcessor, this is basically a TipSetProcessor with no models
type BuiltinProcessor interface {
	ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error)
}

type StateProcessor struct {
	builtinProcessors map[string]BuiltinProcessor
	tipsetProcessors  map[string]TipSetProcessor
	tipsetsProcessors map[string]TipSetsProcessor
	actorProcessors   map[string]ActorProcessor

	// api used by tasks
	api task.TaskAPI

	// wait group used to signal processors completion
	pwg sync.WaitGroup

	// name of the processor
	name string

	// taskNames is a list of all tasks StateProcessor was instructed to process
	taskNames []string
}

func (sp *StateProcessor) ProcessState(ctx context.Context, current, executed *types.TipSet) chan *ModelResults {
	results := sp.processState(ctx, current, executed)
	return sp.processTaskResults(ctx, current, results)
}

// A TaskResult is either some data to persist or an error which indicates that the task did not complete. Partial
// completions are possible provided the Data contains a persistable log of the results.
type TaskResult struct {
	Task        string
	Error       error
	Report      visormodel.ProcessingReportList
	Data        model.Persistable
	StartedAt   time.Time
	CompletedAt time.Time
}

func (sp *StateProcessor) processState(ctx context.Context, current, executed *types.TipSet) chan *TaskResult {
	rs := len(sp.tipsetProcessors) + len(sp.actorProcessors) + len(sp.tipsetsProcessors) + len(sp.builtinProcessors)
	results := make(chan *TaskResult, rs)

	sp.startBuiltinProcessors(ctx, current, results)
	sp.startTipSetProcessors(ctx, current, results)
	sp.startTipSetsProcessors(ctx, current, executed, results)
	sp.startActorProcessors(ctx, current, executed, results)

	go func() {
		sp.pwg.Wait()
		close(results)
	}()
	return results

}

func (sp *StateProcessor) processTaskResults(ctx context.Context, current *types.TipSet, results chan *TaskResult) chan *ModelResults {
	var (
		out       = make(chan *ModelResults, len(sp.tipsetProcessors)+len(sp.actorProcessors)+len(sp.tipsetsProcessors))
		completed = map[string]struct{}{}
	)

	go func() {
		defer close(out)
		for res := range results {
			select {
			case <-ctx.Done():
				// loop over all tasks expected to complete, if they have not been completed mark them as skipped.
				skipTime := time.Now()
				for _, name := range sp.taskNames {
					if _, complete := completed[name]; !complete {
						log.Debugw("task skipped", "task", name, "reason", "indexer not ready")
						out <- &ModelResults{
							Name:  name,
							Model: model.PersistableList{sp.buildSkippedReport(current, name, skipTime, "indexer not ready")},
						}
					}
				}
				stats.Record(ctx, metrics.TipSetSkip.M(1))
				return
			default:
			}

			llt := log.With("task", res.Task)

			// Was there a fatal error?
			if res.Error != nil {
				llt.Errorw("task returned with error", "error", res.Error.Error())
				return
			}

			if res.Report == nil || len(res.Report) == 0 {
				// Nothing was done for this tipset
				llt.Debugw("task returned with no report")
				continue
			}

			for idx := range res.Report {
				// Fill in some report metadata
				res.Report[idx].Reporter = sp.name
				res.Report[idx].Task = res.Task
				res.Report[idx].StartedAt = res.StartedAt
				res.Report[idx].CompletedAt = res.CompletedAt

				if res.Report[idx].ErrorsDetected != nil {
					res.Report[idx].Status = visormodel.ProcessingStatusError
				} else if res.Report[idx].StatusInformation != "" {
					res.Report[idx].Status = visormodel.ProcessingStatusInfo
				} else {
					res.Report[idx].Status = visormodel.ProcessingStatusOK
				}

				llt.Debugw("task report", "status", res.Report[idx].Status, "duration", res.Report[idx].CompletedAt.Sub(res.Report[idx].StartedAt))
			}

			// Persist the processing report and the data in a single transaction
			out <- &ModelResults{
				Name:  res.Task,
				Model: model.PersistableList{res.Report, res.Data},
			}
			completed[res.Task] = struct{}{}
		}
	}()

	return out
}

func (sp *StateProcessor) startBuiltinProcessors(ctx context.Context, current *types.TipSet, results chan *TaskResult) {
	start := time.Now()
	for taskName, proc := range sp.builtinProcessors {
		name := taskName
		p := proc

		sp.pwg.Add(1)
		go func() {
			defer sp.pwg.Done()

			report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &TaskResult{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				return
			}
			results <- &TaskResult{
				Task:        name,
				Report:      report,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
}

func (sp *StateProcessor) startTipSetProcessors(ctx context.Context, current *types.TipSet, results chan *TaskResult) {
	start := time.Now()
	for taskName, proc := range sp.tipsetProcessors {
		name := taskName
		p := proc

		sp.pwg.Add(1)
		go func() {
			defer sp.pwg.Done()

			data, report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &TaskResult{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				return
			}
			results <- &TaskResult{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
}

func (sp *StateProcessor) startTipSetsProcessors(ctx context.Context, current, executed *types.TipSet, results chan *TaskResult) {
	start := time.Now()
	for taskName, proc := range sp.tipsetsProcessors {
		name := taskName
		p := proc

		sp.pwg.Add(1)
		go func() {
			defer sp.pwg.Done()

			data, report, err := p.ProcessTipSets(ctx, current, executed)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &TaskResult{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				return
			}
			results <- &TaskResult{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
}

func (sp *StateProcessor) startActorProcessors(ctx context.Context, current, executed *types.TipSet, results chan *TaskResult) {
	if len(sp.actorProcessors) == 0 {
		return
	}

	changesStart := time.Now()
	changes, err := sp.api.StateChangedActors(ctx, sp.api.Store(), executed, current)
	if err != nil {
		// report all processor tasks as failed
		for name := range sp.actorProcessors {
			results <- &TaskResult{
				Task:  name,
				Error: nil,
				Report: visormodel.ProcessingReportList{&visormodel.ProcessingReport{
					Height:         int64(executed.Height()),
					StateRoot:      executed.ParentState().String(),
					Reporter:       sp.name,
					Task:           name,
					StartedAt:      changesStart,
					CompletedAt:    time.Now(),
					Status:         visormodel.ProcessingStatusError,
					ErrorsDetected: err,
				}},
				Data:        nil,
				StartedAt:   changesStart,
				CompletedAt: time.Now(),
			}
		}
		return
	}

	start := time.Now()
	for taskName, proc := range sp.actorProcessors {
		name := taskName
		p := proc

		sp.pwg.Add(1)
		go func() {
			defer sp.pwg.Done()

			data, report, err := p.ProcessActors(ctx, current, executed, changes)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &TaskResult{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				return
			}
			results <- &TaskResult{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
}

func (sp *StateProcessor) buildSkippedReport(ts *types.TipSet, taskName string, timestamp time.Time, reason string) *visormodel.ProcessingReport {
	return &visormodel.ProcessingReport{
		Height:            int64(ts.Height()),
		StateRoot:         ts.ParentState().String(),
		Reporter:          sp.name,
		Task:              taskName,
		StartedAt:         timestamp,
		CompletedAt:       timestamp,
		Status:            visormodel.ProcessingStatusSkip,
		StatusInformation: reason,
	}
}
