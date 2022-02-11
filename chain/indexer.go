package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/lens/task"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/blocks"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
	"github.com/filecoin-project/lily/tasks/indexer"
	"github.com/filecoin-project/lily/tasks/messageexecutions"
	"github.com/filecoin-project/lily/tasks/messages"
	"github.com/filecoin-project/lily/tasks/msapprovals"
)

const (
	ActorStatesRawTask      = "actorstatesraw"      // task that only extracts raw actor state
	ActorStatesPowerTask    = "actorstatespower"    // task that only extracts power actor states (but not the raw state)
	ActorStatesRewardTask   = "actorstatesreward"   // task that only extracts reward actor states (but not the raw state)
	ActorStatesMinerTask    = "actorstatesminer"    // task that only extracts miner actor states (but not the raw state)
	ActorStatesInitTask     = "actorstatesinit"     // task that only extracts init actor states (but not the raw state)
	ActorStatesMarketTask   = "actorstatesmarket"   // task that only extracts market actor states (but not the raw state)
	ActorStatesMultisigTask = "actorstatesmultisig" // task that only extracts multisig actor states (but not the raw state)
	ActorStatesVerifreg     = "actorstatesverifreg" // task that only extracts verified registry actor states (but not the raw state)
	BlocksTask              = "blocks"              // task that extracts block data
	MessagesTask            = "messages"            // task that extracts message data
	ChainEconomicsTask      = "chaineconomics"      // task that extracts chain economics data
	MultisigApprovalsTask   = "msapprovals"         // task that extracts multisig actor approvals
	ImplicitMessageTask     = "implicitmessage"     // task that extract implicitly executed messages: cron tick and block reward.
	ChainConsensusTask      = "consensus"
)

var AllTasks = []string{
	ActorStatesRawTask,
	ActorStatesPowerTask,
	ActorStatesRewardTask,
	ActorStatesMinerTask,
	ActorStatesInitTask,
	ActorStatesMarketTask,
	ActorStatesMultisigTask,
	ActorStatesVerifreg,
	BlocksTask,
	MessagesTask,
	ChainEconomicsTask,
	MultisigApprovalsTask,
	ImplicitMessageTask,
	ChainConsensusTask,
}

var log = logging.Logger("lily/chain")

var _ TipSetObserver = (*TipSetIndexer)(nil)

// TipSetIndexer waits for tipsets and persists their block data into a database.
type TipSetIndexer struct {
	window            time.Duration
	storage           model.Storage
	builtinProcessors map[string]BuiltinProcessor
	tipsetProcessors  map[string]TipSetProcessor
	tipsetsProcessors map[string]TipSetsProcessor
	actorProcessors   map[string]ActorProcessor
	name              string
	persistSlot       chan struct{} // filled with a token when a goroutine is persisting data
	node              task.TaskAPI
	tasks             []string
	inFlightTasks     int
}

type TipSetIndexerOpt func(t *TipSetIndexer)

// NewTipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(node task.TaskAPI, d model.Storage, window time.Duration, name string, tasks []string, options ...TipSetIndexerOpt) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		storage:           d,
		window:            window,
		name:              name,
		persistSlot:       make(chan struct{}, 1), // allow one concurrent persistence job
		builtinProcessors: map[string]BuiltinProcessor{},
		tipsetProcessors:  map[string]TipSetProcessor{},
		tipsetsProcessors: map[string]TipSetsProcessor{},
		actorProcessors:   map[string]ActorProcessor{},
		node:              node,
		tasks:             tasks,
	}

	// add the builtin processors
	// TODO you can be more specific and call this the null round processor or something.
	tsi.builtinProcessors["builtin"] = indexer.NewTask(node)

	for _, t := range tasks {
		switch t {
		case BlocksTask:
			tsi.tipsetProcessors[BlocksTask] = blocks.NewTask()
		case ChainEconomicsTask:
			tsi.tipsetProcessors[ChainEconomicsTask] = chaineconomics.NewTask(node)
		case ChainConsensusTask:
			tsi.tipsetProcessors[ChainConsensusTask] = consensus.NewTask(node)

		case ActorStatesRawTask:
			tsi.actorProcessors[ActorStatesRawTask] = actorstate.NewTask(node, &actorstate.RawActorExtractorMap{})
		case ActorStatesPowerTask:
			tsi.actorProcessors[ActorStatesPowerTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(power.AllCodes()))
		case ActorStatesRewardTask:
			tsi.actorProcessors[ActorStatesRewardTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(reward.AllCodes()))
		case ActorStatesMinerTask:
			tsi.actorProcessors[ActorStatesMinerTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(miner.AllCodes()))
		case ActorStatesInitTask:
			tsi.actorProcessors[ActorStatesInitTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(init_.AllCodes()))
		case ActorStatesMarketTask:
			tsi.actorProcessors[ActorStatesMarketTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(market.AllCodes()))
		case ActorStatesMultisigTask:
			tsi.actorProcessors[ActorStatesMultisigTask] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(multisig.AllCodes()))
		case ActorStatesVerifreg:
			tsi.actorProcessors[ActorStatesVerifreg] = actorstate.NewTask(node, actorstate.NewTypedActorExtractorMap(verifreg.AllCodes()))

		case MessagesTask:
			tsi.tipsetsProcessors[MessagesTask] = messages.NewTask(node)
		case MultisigApprovalsTask:
			tsi.tipsetsProcessors[MultisigApprovalsTask] = msapprovals.NewTask(node)
		case ImplicitMessageTask:
			tsi.tipsetsProcessors[ImplicitMessageTask] = messageexecutions.NewTask(node)
		default:
			return nil, xerrors.Errorf("unknown task: %s", t)
		}
	}

	for _, opt := range options {
		opt(tsi)
	}

	return tsi, nil
}

// TipSet is called when a new tipset has been discovered
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Name, t.name))
	ctx, span := otel.Tracer("").Start(ctx, "TipSetIndexer.TipSet")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("tipset", ts.String()),
			attribute.Int64("height", int64(ts.Height())),
			attribute.String("name", t.name),
			attribute.String("window", t.window.String()),
			attribute.StringSlice("tasks", t.tasks),
		)
	}
	defer span.End()

	if ts.Height() == 0 {
		// bail, the parent of genesis is itself, there is no diff
		return nil
	}

	var executed, current *types.TipSet
	pts, err := t.node.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return err
	}
	current = ts
	executed = pts

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("current_tipset", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
			attribute.String("executed_tipset", executed.String()),
			attribute.Int64("executed_height", int64(executed.Height())),
		)
	}

	// TODO this should be controlled by the caller of TipSet
	var cancel func()
	var tctx context.Context // cancellable context for the task
	if t.window > 0 {
		// Do as much indexing as possible in the specified time window (usually one epoch when following head of chain)
		// Anything not completed in that time will be marked as incomplete
		tctx, cancel = context.WithTimeout(ctx, t.window)
	} else {
		// Ensure all goroutines are stopped when we exit
		tctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	start := time.Now()

	// "Extract" state form the chain.
	taskResults := t.index(tctx, executed, current)

	// "Transform" extracted state to models.
	modelResults := t.process(tctx, executed, taskResults)

	// "Load" the models into storage.
	if err := t.persist(tctx, modelResults); err != nil {
		return err
	}
	log.Infow("indexed tipsets", "duration", time.Since(start), "executed", int64(executed.Height()), "current", int64(current.Height()), "tipset", ts.Key().String())
	return nil
}

func (t *TipSetIndexer) index(ctx context.Context, executed, current *types.TipSet) chan *TaskResult {
	t.inFlightTasks = len(t.tipsetProcessors) + len(t.actorProcessors) + len(t.tipsetsProcessors) + len(t.builtinProcessors)
	results := make(chan *TaskResult, t.inFlightTasks)

	t.startBuiltinProcessors(ctx, t.builtinProcessors, current, results)

	t.startProcessors(ctx, t.tipsetProcessors, current, results)

	t.startMessageProcessors(ctx, t.tipsetsProcessors, executed, current, results)

	t.startActorProcessors(ctx, t.actorProcessors, executed, current, results)

	return results
}

type ModelResults struct {
	Name  string
	Model model.PersistableList
}

func (t *TipSetIndexer) process(ctx context.Context, ts *types.TipSet, results chan *TaskResult) chan *ModelResults {
	var (
		out       = make(chan *ModelResults, len(t.tipsetProcessors)+len(t.actorProcessors)+len(t.tipsetsProcessors))
		completed = map[string]struct{}{}
	)

	go func() {
		defer close(out)
		for t.inFlightTasks > 0 {
			var res *TaskResult
			select {
			case <-ctx.Done():
				// if the indexers timeout (window) context is done then we have run out of time.
				// loop over all tasks expected to complete, if they have not been completed mark them as skipped.
				skipTime := time.Now()
				for _, name := range t.tasks {
					if _, complete := completed[name]; !complete {
						log.Debugw("task skipped", "task", name, "reason", "indexer not ready")
						out <- &ModelResults{
							Name:  name,
							Model: model.PersistableList{t.buildSkippedTipsetReport(ts, name, skipTime, "indexer not ready")},
						}
					}
				}
				stats.Record(ctx, metrics.TipSetSkip.M(1))
				return
			case res = <-results:
				t.inFlightTasks--

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
					res.Report[idx].Reporter = t.name
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
		}
	}()

	return out
}

func (t *TipSetIndexer) persist(ctx context.Context, models chan *ModelResults) error {
	for res := range models {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t.persistSlot <- struct{}{}:
		}
		// wait until there is an empty slot before persisting
		start := time.Now()
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, res.Name))

		if err := t.storage.PersistBatch(ctx, res.Model); err != nil {
			stats.Record(ctx, metrics.PersistFailure.M(1))
			log.Errorw("persistence failed", "task", res.Name, "error", err)
			return err
		}
		log.Debugw("task data persisted", "task", res.Model, "duration", time.Since(start))
		<-t.persistSlot
	}

	return nil
}

func (t *TipSetIndexer) startProcessors(ctx context.Context, processors map[string]TipSetProcessor, current *types.TipSet, results chan *TaskResult) {
	for name, p := range processors {
		log.Debugw("starting processor", "name", name)
		go t.runProcessor(ctx, p, name, current, results)
	}
}

func (t *TipSetIndexer) startMessageProcessors(ctx context.Context, processors map[string]TipSetsProcessor, executed, current *types.TipSet, results chan *TaskResult) {
	for name, p := range processors {
		log.Debugw("starting processor", "name", name)
		go t.runMessageProcessor(ctx, p, name, current, executed, results)
	}
}

func (t *TipSetIndexer) startActorProcessors(ctx context.Context, processors map[string]ActorProcessor, executed, current *types.TipSet, results chan *TaskResult) {
	// If we have actor processors then find actors that have changed state
	if len(processors) > 0 {
		changesStart := time.Now()
		changes, err := t.node.StateChangedActors(ctx, t.node.Store(), executed, current)
		if err != nil {
			// report all processor tasks as failed
			for name := range processors {
				results <- &TaskResult{
					Task:  name,
					Error: nil,
					Report: visormodel.ProcessingReportList{&visormodel.ProcessingReport{
						Height:         int64(executed.Height()),
						StateRoot:      executed.ParentState().String(),
						Reporter:       t.name,
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

		log.Debugw("found actor state changes", "count", len(changes), "time", time.Since(changesStart))
		for name, p := range t.actorProcessors {
			go t.runActorProcessor(ctx, p, name, current, executed, changes, results)
		}
	}
}

func (t *TipSetIndexer) startBuiltinProcessors(ctx context.Context, processors map[string]BuiltinProcessor, current *types.TipSet, results chan *TaskResult) {
	for name, p := range processors {
		log.Debugw("starting processor", "name", name)
		go t.runBuiltinProcessor(ctx, p, name, current, results)
	}
}

func (t *TipSetIndexer) runBuiltinProcessor(ctx context.Context, p BuiltinProcessor, name string, ts *types.TipSet, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.Processor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	report, err := p.ProcessTipSet(ctx, ts)
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

}

func (t *TipSetIndexer) runProcessor(ctx context.Context, p TipSetProcessor, name string, ts *types.TipSet, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.Processor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessTipSet(ctx, ts)
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
}

func (t *TipSetIndexer) runMessageProcessor(ctx context.Context, p TipSetsProcessor, name string, ts, pts *types.TipSet, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.TipSetsProcessor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessTipSets(ctx, ts, pts)
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
}

func (t *TipSetIndexer) runActorProcessor(ctx context.Context, p ActorProcessor, name string, ts, pts *types.TipSet, actors task.ActorStateChangeDiff, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.ActorProcessor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessActors(ctx, ts, pts, actors)
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
}

func (t *TipSetIndexer) Close() error {
	log.Debug("closing tipset indexer")

	// We need to ensure that any persistence goroutine has completed. Since the channel has capacity 1 we can detect
	// when the persistence goroutine is running by attempting to send a probe value on the channel. When the channel
	// contains a token then we are still persisting and we should wait for that to be done.
	select {
	case t.persistSlot <- struct{}{}:
		// no token was in channel so there was no persistence goroutine running
	default:
		// channel contained a token so persistence goroutine is running
		// wait for the persistence to finish, which is when the channel can be sent on
		log.Debug("waiting for persistence to complete")
		t.persistSlot <- struct{}{}
		log.Debug("persistence completed")
	}

	// When we reach here there will always be a single token in the channel (our probe) which needs to be drained so
	// the channel is empty for reuse.
	<-t.persistSlot

	return nil
}

func (t *TipSetIndexer) buildSkippedTipsetReport(ts *types.TipSet, taskName string, timestamp time.Time, reason string) *visormodel.ProcessingReport {
	return &visormodel.ProcessingReport{
		Height:            int64(ts.Height()),
		StateRoot:         ts.ParentState().String(),
		Reporter:          t.name,
		Task:              taskName,
		StartedAt:         timestamp,
		CompletedAt:       timestamp,
		Status:            visormodel.ProcessingStatusSkip,
		StatusInformation: reason,
	}
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
