package chain

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

const (
	ActorStatesRawTask      = "actorstatesraw"      // task that only extracts raw actor state
	ActorStatesPowerTask    = "actorstatespower"    // task that only extracts power actor states (but not the raw state)
	ActorStatesRewardTask   = "actorstatesreward"   // task that only extracts reward actor states (but not the raw state)
	ActorStatesMinerTask    = "actorstatesminer"    // task that only extracts miner actor states (but not the raw state)
	ActorStatesInitTask     = "actorstatesinit"     // task that only extracts init actor states (but not the raw state)
	ActorStatesMarketTask   = "actorstatesmarket"   // task that only extracts market actor states (but not the raw state)
	ActorStatesMultisigTask = "actorstatesmultisig" // task that only extracts multisig actor states (but not the raw state)
	BlocksTask              = "blocks"              // task that extracts block data
	MessagesTask            = "messages"            // task that extracts message data
	ChainEconomicsTask      = "chaineconomics"      // task that extracts chain economics data
)

var log = logging.Logger("chain")

var _ TipSetObserver = (*TipSetIndexer)(nil)

// A TipSetWatcher waits for tipsets and persists their block data into a database.
type TipSetIndexer struct {
	window          time.Duration
	storage         model.Storage
	processors      map[string]TipSetProcessor
	actorProcessors map[string]ActorProcessor
	name            string
	persistSlot     chan struct{}
	lastTipSet      *types.TipSet
	node            lens.API
	opener          lens.APIOpener
	closer          lens.APICloser
}

// A TipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(o lens.APIOpener, d model.Storage, window time.Duration, name string, tasks []string) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		storage:         d,
		window:          window,
		name:            name,
		persistSlot:     make(chan struct{}, 1), // allow one concurrent persistence job
		processors:      map[string]TipSetProcessor{},
		actorProcessors: map[string]ActorProcessor{},
		opener:          o,
	}

	for _, task := range tasks {
		switch task {
		case BlocksTask:
			tsi.processors[BlocksTask] = NewBlockProcessor()
		case MessagesTask:
			tsi.processors[MessagesTask] = NewMessageProcessor(o)
		case ChainEconomicsTask:
			tsi.processors[ChainEconomicsTask] = NewChainEconomicsProcessor(o)
		case ActorStatesRawTask:
			tsi.actorProcessors[ActorStatesRawTask] = NewActorStateProcessor(o, &RawActorExtractorMap{})
		case ActorStatesPowerTask:
			tsi.actorProcessors[ActorStatesPowerTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.StoragePowerActorCodeID,
				CodeV2: sa2builtin.StoragePowerActorCodeID,
				CodeV3: sa3builtin.StoragePowerActorCodeID,
			})
		case ActorStatesRewardTask:
			tsi.actorProcessors[ActorStatesRewardTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.RewardActorCodeID,
				CodeV2: sa2builtin.RewardActorCodeID,
				CodeV3: sa3builtin.RewardActorCodeID,
			})
		case ActorStatesMinerTask:
			tsi.actorProcessors[ActorStatesMinerTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.StorageMinerActorCodeID,
				CodeV2: sa2builtin.StorageMinerActorCodeID,
				CodeV3: sa3builtin.StorageMinerActorCodeID,
			})
		case ActorStatesInitTask:
			tsi.actorProcessors[ActorStatesInitTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.InitActorCodeID,
				CodeV2: sa2builtin.InitActorCodeID,
				CodeV3: sa3builtin.InitActorCodeID,
			})
		case ActorStatesMarketTask:
			tsi.actorProcessors[ActorStatesMarketTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.StorageMarketActorCodeID,
				CodeV2: sa2builtin.StorageMarketActorCodeID,
				CodeV3: sa3builtin.StorageMarketActorCodeID,
			})
		case ActorStatesMultisigTask:
			tsi.actorProcessors[ActorStatesMultisigTask] = NewActorStateProcessor(o, &TypedActorExtractorMap{
				CodeV1: sa0builtin.MultisigActorCodeID,
				CodeV2: sa2builtin.MultisigActorCodeID,
				CodeV3: sa3builtin.MultisigActorCodeID,
			})
		default:
			return nil, xerrors.Errorf("unknown task: %s", task)
		}
	}
	return tsi, nil
}

// TipSet is called when a new tipset has been discovered
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Name, t.name))

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

	start := time.Now()

	inFlight := 0
	results := make(chan *TaskResult, len(t.processors)+len(t.actorProcessors))

	// A map to gather the persistable outputs from each task
	taskOutputs := make(map[string]model.PersistableList, len(t.processors)+len(t.actorProcessors))

	// Run each tipset processing task concurrently
	for name, p := range t.processors {
		inFlight++
		go t.runProcessor(tctx, p, name, ts, results)
	}

	// Run each actor processing task concurrently if we have any and we've seen a previous tipset to compare with
	if len(t.actorProcessors) > 0 {

		// Actor processors perform a diff between two tipsets so we need to keep track of parent and child
		var parent, child *types.TipSet
		if t.lastTipSet != nil {
			if t.lastTipSet.Height() > ts.Height() {
				// last tipset seen was the child
				child = t.lastTipSet
				parent = ts
			} else if t.lastTipSet.Height() < ts.Height() {
				// last tipset seen was the parent
				child = ts
				parent = t.lastTipSet
			} else {
				log.Errorw("out of order tipsets", "height", ts.Height(), "last_height", t.lastTipSet.Height())
			}
		}

		// If no parent tipset available then we need to skip processing. It's likely we received the last or first tipset
		// in a batch. No report is generated because a different run of the indexer could cover the parent and child
		// for this tipset.
		if parent != nil {
			if t.node == nil {
				node, closer, err := t.opener.Open(ctx)
				if err != nil {
					return xerrors.Errorf("unable to open lens: %w", err)
				}
				t.node = node
				t.closer = closer
			}

			changes, err := t.node.StateChangedActors(tctx, parent.ParentState(), child.ParentState())
			if err != nil {

				terr := xerrors.Errorf("failed to extract actor changes: %w", err)
				// We need to report that all actor tasks failed
				for name := range t.actorProcessors {
					report := &visormodel.ProcessingReport{
						Height:         int64(ts.Height()),
						StateRoot:      ts.ParentState().String(),
						Reporter:       t.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: terr,
					}
					taskOutputs[name] = model.PersistableList{report}
				}
				return terr
			}

			for name, p := range t.actorProcessors {
				inFlight++
				go t.runActorProcessor(tctx, p, name, child, parent, changes, results)
			}
		}
	}

	ll := log.With("height", int64(ts.Height()))

	// Wait for all tasks to complete
	for inFlight > 0 {
		res := <-results
		inFlight--

		llt := ll.With("task", res.Task)

		// Was there a fatal error?
		if res.Error != nil {
			llt.Errorw("task returned with error", "error", res.Error.Error())
			// tell all the processors to close their connections to the lens, they can reopen when needed
			if err := t.Close(); err != nil {
				log.Errorw("error received while closing tipset indexer", "error", err)
			}
			return res.Error
		}

		if res.Report == nil {
			// Nothing was done for this tipset
			llt.Debugw("task returned with no report")
			continue
		}

		// Fill in some report metadata
		res.Report.Reporter = t.name
		res.Report.Task = res.Task
		res.Report.StartedAt = start
		res.Report.CompletedAt = time.Now()

		if res.Report.ErrorsDetected != nil {
			res.Report.Status = visormodel.ProcessingStatusError
		} else if res.Report.StatusInformation != "" {
			res.Report.Status = visormodel.ProcessingStatusInfo
		} else {
			res.Report.Status = visormodel.ProcessingStatusOK
		}

		llt.Infow("task report", "status", res.Report.Status, "time", res.Report.CompletedAt.Sub(res.Report.StartedAt))

		// Persist the processing report and the data in a single transaction
		taskOutputs[res.Task] = model.PersistableList{res.Report, res.Data}
	}

	// remember the last tipset we observed
	t.lastTipSet = ts

	if len(taskOutputs) == 0 {
		// Nothing to persist
		ll.Debugw("tipset complete, nothing to persist", "total_time", time.Since(start))
		return nil
	}

	// wait until there is an empty slot before persisting
	ll.Debugw("waiting to persist data", "time", time.Since(start))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.persistSlot <- struct{}{}:
		// Slot is free so we can continue
	}

	// Persist all results
	go func() {
		// free up the slot when done
		defer func() {
			<-t.persistSlot
		}()

		ll.Debugw("persisting data", "time", time.Since(start))
		var wg sync.WaitGroup
		wg.Add(len(taskOutputs))

		// Persist each processor's data concurrently since they don't overlap
		for task, p := range taskOutputs {
			go func(task string, p model.Persistable) {
				defer wg.Done()
				ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, task))

				if err := t.storage.PersistBatch(ctx, p); err != nil {
					stats.Record(ctx, metrics.PersistFailure.M(1))
					ll.Errorw("persistence failed", "task", task, "error", err)
					return
				}
				ll.Debugw("task data persisted", "task", task, "time", time.Since(start))
			}(task, p)
		}
		wg.Wait()
		ll.Debugw("tipset complete", "total_time", time.Since(start))
	}()

	return nil
}

func (t *TipSetIndexer) runProcessor(ctx context.Context, p TipSetProcessor, name string, ts *types.TipSet, results chan *TaskResult) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	data, report, err := p.ProcessTipSet(ctx, ts)
	if err != nil {
		stats.Record(ctx, metrics.ProcessingFailure.M(1))
		results <- &TaskResult{
			Task:  name,
			Error: err,
		}
		return
	}
	results <- &TaskResult{
		Task:   name,
		Report: report,
		Data:   data,
	}
}

func (t *TipSetIndexer) runActorProcessor(ctx context.Context, p ActorProcessor, name string, ts, pts *types.TipSet, actors map[string]types.Actor, results chan *TaskResult) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	data, report, err := p.ProcessActors(ctx, ts, pts, actors)
	if err != nil {
		stats.Record(ctx, metrics.ProcessingFailure.M(1))
		results <- &TaskResult{
			Task:  name,
			Error: err,
		}
		return
	}
	results <- &TaskResult{
		Task:   name,
		Report: report,
		Data:   data,
	}
}

func (t *TipSetIndexer) Close() error {
	if t.closer != nil {
		t.closer()
		t.closer = nil
	}
	t.node = nil

	for name, p := range t.processors {
		if err := p.Close(); err != nil {
			log.Errorw("error received while closing task processor", "error", err, "task", name)
		}
	}
	for name, p := range t.actorProcessors {
		if err := p.Close(); err != nil {
			log.Errorw("error received while closing actor task processor", "error", err, "task", name)
		}
	}
	return nil
}

// A TaskResult is either some data to persist or an error which indicates that the task did not complete. Partial
// completions are possible provided the Data contains a persistable log of the results.
type TaskResult struct {
	Task   string
	Error  error
	Report *visormodel.ProcessingReport
	Data   model.Persistable
}

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
	Close() error
}

type ActorProcessor interface {
	// ProcessActor processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessActors(ctx context.Context, ts *types.TipSet, pts *types.TipSet, actors map[string]types.Actor) (model.Persistable, *visormodel.ProcessingReport, error)
	Close() error
}
