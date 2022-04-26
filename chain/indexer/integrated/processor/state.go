package processor

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks"
)

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type TipSetsProcessor interface {
	// ProcessTipSets processes sequential tipsts (a parent and a child, or an executed and a current). If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSets(ctx context.Context, current *types.TipSet, executed *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessActors(ctx context.Context, current *types.TipSet, executed *types.TipSet, actors tasks.ActorStateChangeDiff) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ReportProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Implementations of this interface must abort processing when their context is canceled.
	ProcessTipSet(ctx context.Context, current *types.TipSet) (visormodel.ProcessingReportList, error)
}

var log = logging.Logger("lily/index/processor")

type StateProcessor struct {
	builtinProcessors map[string]ReportProcessor
	tipsetProcessors  map[string]TipSetProcessor
	tipsetsProcessors map[string]TipSetsProcessor
	actorProcessors   map[string]ActorProcessor

	// api used by tasks
	api tasks.DataSource

	//pwg is a wait group used internally to signal processors completion
	pwg sync.WaitGroup

	// name of the processor
	name string
}

// A Result is either some data to persist or an error which indicates that the task did not complete. Partial
// completions are possibly provided the Data contains a persistable log of the results.
type Result struct {
	Task        string
	Error       error
	Report      visormodel.ProcessingReportList
	Data        model.Persistable
	StartedAt   time.Time
	CompletedAt time.Time
}

// State executes its configured processors in parallel, processing the state in `current` and `executed. The return channel
// emits results of the state extraction closing when processing is completed. It is the responsibility of the processors
// to abort if its context is canceled.
// A list of all tasks executing is returned.
func (sp *StateProcessor) State(ctx context.Context, current, executed *types.TipSet) (chan *Result, []string) {
	ctx, span := otel.Tracer("").Start(ctx, "StateProcessor.State")

	num := len(sp.tipsetProcessors) + len(sp.actorProcessors) + len(sp.tipsetsProcessors) + len(sp.builtinProcessors)
	results := make(chan *Result, num)
	taskNames := make([]string, 0, num)

	taskNames = append(taskNames, sp.startReport(ctx, current, results)...)
	taskNames = append(taskNames, sp.startTipSet(ctx, current, results)...)
	taskNames = append(taskNames, sp.startTipSets(ctx, current, executed, results)...)
	taskNames = append(taskNames, sp.startActor(ctx, current, executed, results)...)

	go func() {
		sp.pwg.Wait()
		defer span.End()
		close(results)
	}()
	return results, taskNames
}

// startReport starts all ReportProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startReport(ctx context.Context, current *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.builtinProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      report,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startTipSet starts all TipSetProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startTipSet(ctx context.Context, current *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.tipsetProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			data, report, err := p.ProcessTipSet(ctx, current)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startTipSets starts all TipSetsProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startTipSets(ctx context.Context, current, executed *types.TipSet, results chan *Result) []string {
	start := time.Now()
	var taskNames []string
	for taskName, proc := range sp.tipsetsProcessors {
		name := taskName
		p := proc
		taskNames = append(taskNames, name)

		sp.pwg.Add(1)
		go func() {
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
			stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
			stop := metrics.Timer(ctx, metrics.ProcessingDuration)
			defer stop()

			pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
			pl.Infow("processor started")
			defer func() {
				pl.Infow("processor ended", "duration", time.Since(start))
				sp.pwg.Done()
			}()

			data, report, err := p.ProcessTipSets(ctx, current, executed)
			if err != nil {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:        name,
					Error:       err,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				pl.Errorw("processor error", "error", err)
				return
			}
			results <- &Result{
				Task:        name,
				Report:      visormodel.ProcessingReportList{report},
				Data:        data,
				StartedAt:   start,
				CompletedAt: time.Now(),
			}
		}()
	}
	return taskNames
}

// startActor starts all ActorProcessor's in parallel, their results are emitted on the `results` channel.
// A list containing all executed task names is returned.
func (sp *StateProcessor) startActor(ctx context.Context, current, executed *types.TipSet, results chan *Result) []string {

	if len(sp.actorProcessors) == 0 {
		return nil
	}

	var taskNames []string
	for name := range sp.actorProcessors {
		taskNames = append(taskNames, name)
	}

	sp.pwg.Add(len(sp.actorProcessors))
	go func() {
		start := time.Now()
		changes, err := sp.api.ActorStateChanges(ctx, current, executed)
		if err != nil {
			// report all processor tasks as failed
			for name := range sp.actorProcessors {
				stats.Record(ctx, metrics.ProcessingFailure.M(1))
				results <- &Result{
					Task:  name,
					Error: nil,
					Report: visormodel.ProcessingReportList{&visormodel.ProcessingReport{
						Height:         int64(current.Height()),
						StateRoot:      current.ParentState().String(),
						Reporter:       sp.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: err,
					}},
					Data:        nil,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
				sp.pwg.Done()
			}
			return
		}

		for taskName, proc := range sp.actorProcessors {
			name := taskName
			p := proc

			go func() {
				ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
				stats.Record(ctx, metrics.TipsetHeight.M(int64(current.Height())))
				stop := metrics.Timer(ctx, metrics.ProcessingDuration)
				defer stop()

				pl := log.With("task", name, "height", current.Height(), "reporter", sp.name)
				pl.Infow("processor started")
				defer func() {
					pl.Infow("processor ended", "duration", time.Since(start))
					sp.pwg.Done()
				}()

				data, report, err := p.ProcessActors(ctx, current, executed, changes)
				if err != nil {
					stats.Record(ctx, metrics.ProcessingFailure.M(1))
					results <- &Result{
						Task:        name,
						Error:       err,
						StartedAt:   start,
						CompletedAt: time.Now(),
					}
					pl.Warnw("processor error", "error", err)
					return
				}
				results <- &Result{
					Task:        name,
					Report:      visormodel.ProcessingReportList{report},
					Data:        data,
					StartedAt:   start,
					CompletedAt: time.Now(),
				}
			}()
		}
	}()
	return taskNames
}
