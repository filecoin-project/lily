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
)

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

	// complete is a map of tasks and their completeness.
	complete   map[string]bool
	completeMu sync.Mutex
}

func (sp *StateProcessor) ProcessState(ctx context.Context, current, executed *types.TipSet) chan *TaskResult {
	// build the taskName list
	for name := range sp.builtinProcessors {
		sp.taskNames = append(sp.taskNames, name)
		sp.complete[name] = false
	}
	for name := range sp.tipsetProcessors {
		sp.taskNames = append(sp.taskNames, name)
		sp.complete[name] = false
	}
	for name := range sp.tipsetsProcessors {
		sp.taskNames = append(sp.taskNames, name)
		sp.complete[name] = false
	}
	for name := range sp.actorProcessors {
		sp.taskNames = append(sp.taskNames, name)
		sp.complete[name] = false
	}

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

func (sp *StateProcessor) startBuiltinProcessors(ctx context.Context, current *types.TipSet, results chan *TaskResult) {
	start := time.Now()
	for taskName, proc := range sp.builtinProcessors {
		name := taskName
		p := proc

		sp.pwg.Add(1)
		go func() {
			defer func() {
				sp.taskComplete(name)
				sp.pwg.Done()
			}()

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
			defer func() {
				sp.taskComplete(name)
				sp.pwg.Done()
			}()

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
			defer func() {
				sp.taskComplete(name)
				sp.pwg.Done()
			}()

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
			defer func() {
				sp.taskComplete(name)
				sp.pwg.Done()
			}()

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

func (sp *StateProcessor) taskComplete(name string) {
	sp.completeMu.Lock()
	sp.complete[name] = true
	sp.completeMu.Unlock()
}

func (sp *StateProcessor) IncompleteTasks() []string {
	sp.completeMu.Lock()
	defer sp.completeMu.Unlock()
	var out []string
	for t, complete := range sp.complete {
		if !complete {
			out = append(out, t)
		}
	}
	return out
}
