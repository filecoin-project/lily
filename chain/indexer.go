package chain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/adt"
	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/blocks"
	"github.com/filecoin-project/lily/tasks/chaineconomics"
	"github.com/filecoin-project/lily/tasks/consensus"
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
	window                     time.Duration
	storage                    model.Storage
	processors                 map[string]TipSetProcessor
	messageProcessors          map[string]MessageProcessor
	messageExecutionProcessors map[string]MessageExecutionProcessor
	actorProcessors            map[string]ActorProcessor
	consensusProcessor         map[string]TipSetsProcessor
	name                       string
	persistSlot                chan struct{} // filled with a token when a goroutine is persisting data
	lastTipSet                 *types.TipSet
	node                       lens.API
	tasks                      []string
}

type TipSetIndexerOpt func(t *TipSetIndexer)

// NewTipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(node lens.API, d model.Storage, window time.Duration, name string, tasks []string, options ...TipSetIndexerOpt) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		storage:                    d,
		window:                     window,
		name:                       name,
		persistSlot:                make(chan struct{}, 1), // allow one concurrent persistence job
		processors:                 map[string]TipSetProcessor{},
		messageProcessors:          map[string]MessageProcessor{},
		messageExecutionProcessors: map[string]MessageExecutionProcessor{},
		consensusProcessor:         map[string]TipSetsProcessor{},
		actorProcessors:            map[string]ActorProcessor{},
		node:                       node,
		tasks:                      tasks,
	}

	for _, task := range tasks {
		switch task {
		case BlocksTask:
			tsi.processors[BlocksTask] = blocks.NewTask()
		case MessagesTask:
			tsi.messageProcessors[MessagesTask] = messages.NewTask()
		case ChainEconomicsTask:
			tsi.processors[ChainEconomicsTask] = chaineconomics.NewTask(node)
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
		case MultisigApprovalsTask:
			tsi.messageProcessors[MultisigApprovalsTask] = msapprovals.NewTask(node)
		case ChainConsensusTask:
			tsi.consensusProcessor[ChainConsensusTask] = consensus.NewTask()
		case ImplicitMessageTask:
			tsi.messageExecutionProcessors[ImplicitMessageTask] = messageexecutions.NewTask()
		default:
			return nil, xerrors.Errorf("unknown task: %s", task)
		}
	}

	for _, opt := range options {
		opt(tsi)
	}

	return tsi, nil
}

// TipSet is called when a new tipset has been discovered
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
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
	// TODO should these be allocated to the size of message and message execution processors
	results := make(chan *TaskResult, len(t.processors)+len(t.actorProcessors)+len(t.messageExecutionProcessors)+len(t.consensusProcessor)+len(t.messageProcessors))
	// A map to gather the persistable outputs from each task
	taskOutputs := make(map[string]model.PersistableList, len(t.processors)+len(t.actorProcessors)+len(t.messageExecutionProcessors)+len(t.consensusProcessor)+len(t.messageProcessors))

	// current is the primary tipset that tasks act upon.
	// next adds additional context such as outcomes of message execution.
	var current, next *types.TipSet
	if t.lastTipSet != nil {
		if t.lastTipSet.Height() > ts.Height() {
			// last tipset seen was the child
			next = t.lastTipSet
			current = ts
		} else if t.lastTipSet.Height() < ts.Height() {
			// last tipset seen was the parent
			next = ts
			current = t.lastTipSet
		} else {
			log.Errorw("out of order tipsets", "height", ts.Height(), "last_height", t.lastTipSet.Height())
		}
	}

	// remember the last tipset we observed
	t.lastTipSet = ts

	// current is nil when we have only seen the first tipset in our sequence. we should wait for the next one.
	if current == nil {
		return nil
	}

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("next_tipset", next.String()),
			attribute.Int64("next_height", int64(next.Height())),
			attribute.String("current_tipset", current.String()),
			attribute.Int64("current_height", int64(current.Height())),
		)
	}

	ll := log.With("current", int64(current.Height()), "next", int64(next.Height()))
	ll.Debugw("indexing tipset")

	// Run each tipset processing task concurrently
	for name, p := range t.processors {
		inFlight++
		go t.runProcessor(tctx, p, name, current, results)
	}

	if types.CidArrsEqual(next.Parents().Cids(), current.Cids()) {
		if len(t.consensusProcessor) > 0 {
			for name, p := range t.consensusProcessor {
				inFlight++
				go t.runConsensusProcessor(ctx, p, name, next, current, results)
			}
		}
		// If we have message or actor processors then extract the messages and receipts
		if len(t.messageProcessors) > 0 || len(t.actorProcessors) > 0 {
			execMessagesStart := time.Now()
			tsMsgs, err := t.node.GetExecutedAndBlockMessagesForTipset(ctx, next, current)
			if err == nil {
				ll.Debugw("found executed messages", "count", len(tsMsgs.Executed), "time", time.Since(execMessagesStart))

				if len(t.messageProcessors) > 0 {
					// Start all the message processors
					for name, p := range t.messageProcessors {
						inFlight++
						go t.runMessageProcessor(tctx, p, name, next, current, tsMsgs.Executed, tsMsgs.Block, results)
					}
				}

				// If we have actor processors then find actors that have changed state
				if len(t.actorProcessors) > 0 {
					changesStart := time.Now()
					var err error
					var changes map[string]lens.ActorStateChange
					// special case, we want to extract all actor states from the genesis block.
					if current.Height() == 0 {
						changes, err = t.getGenesisActors(ctx)
					} else {
						changes, err = t.stateChangedActors(tctx, current.ParentState(), next.ParentState())
					}
					if err == nil {
						ll.Debugw("found actor state changes", "count", len(changes), "time", time.Since(changesStart))
						for name, p := range t.actorProcessors {
							inFlight++
							go t.runActorProcessor(tctx, p, name, next, current, changes, tsMsgs.Executed, results)
						}
					} else {
						ll.Errorw("failed to extract actor changes", "error", err)
						terr := xerrors.Errorf("failed to extract actor changes: %w", err)
						// We need to report that all actor tasks failed
						for name := range t.actorProcessors {
							report := &visormodel.ProcessingReport{
								Height:         int64(current.Height()),
								StateRoot:      current.ParentState().String(),
								Reporter:       t.name,
								Task:           name,
								StartedAt:      start,
								CompletedAt:    time.Now(),
								Status:         visormodel.ProcessingStatusError,
								ErrorsDetected: terr,
							}
							taskOutputs[name] = model.PersistableList{report}
						}
					}
				}

			} else {
				ll.Errorw("failed to extract messages", "error", err)
				terr := xerrors.Errorf("failed to extract messages: %w", err)
				// We need to report that all message tasks failed
				for name := range t.messageProcessors {
					report := &visormodel.ProcessingReport{
						Height:         int64(current.Height()),
						StateRoot:      current.ParentState().String(),
						Reporter:       t.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: terr,
					}
					taskOutputs[name] = model.PersistableList{report}
				}
				// We also need to report that all actor tasks failed
				for name := range t.actorProcessors {
					report := &visormodel.ProcessingReport{
						Height:         int64(current.Height()),
						StateRoot:      current.ParentState().String(),
						Reporter:       t.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: terr,
					}
					taskOutputs[name] = model.PersistableList{report}
				}

			}
		}

		// If we have messages execution processors then extract internal messages
		if len(t.messageExecutionProcessors) > 0 {
			execMessagesStart := time.Now()
			iMsgs, err := t.node.GetMessageExecutionsForTipSet(ctx, next, current)
			if err == nil {
				ll.Debugw("found message execution results", "count", len(iMsgs), "time", time.Since(execMessagesStart))
				// Start all the message processors
				for name, p := range t.messageExecutionProcessors {
					inFlight++
					go t.runMessageExecutionProcessor(tctx, p, name, next, current, iMsgs, results)
				}
			} else {
				ll.Errorw("failed to extract messages", "error", err)
				terr := xerrors.Errorf("failed to extract messages: %w", err)
				// We need to report that all message tasks failed
				for name := range t.messageExecutionProcessors {
					report := &visormodel.ProcessingReport{
						Height:         int64(current.Height()),
						StateRoot:      current.ParentState().String(),
						Reporter:       t.name,
						Task:           name,
						StartedAt:      start,
						CompletedAt:    time.Now(),
						Status:         visormodel.ProcessingStatusError,
						ErrorsDetected: terr,
					}
					taskOutputs[name] = model.PersistableList{report}
				}
			}
		}

	} else {
		// TODO: we could fetch the parent stateroot and proceed to index this tipset. However this will be
		// slower and increases the likelihood that we exceed the processing window and cause the next
		// tipset to be skipped completely.
		ll.Errorw("mismatching current and next tipsets", "current_tipset", current.Key(), "next_tipset", next.Key(), "next_parents", next.Parents().Cids())

		// We need to report that all message and actor tasks were skipped
		reason := "tipset did not have expected parent or child"
		for name := range t.messageProcessors {
			taskOutputs[name] = model.PersistableList{t.buildSkippedTipsetReport(ts, name, start, reason)}
			ll.Infow("task skipped", "task", name, "reason", reason)
		}
		for name := range t.messageExecutionProcessors {
			taskOutputs[name] = model.PersistableList{t.buildSkippedTipsetReport(ts, name, start, reason)}
			ll.Infow("task skipped", "task", name, "reason", reason)
		}
		for name := range t.consensusProcessor {
			taskOutputs[name] = model.PersistableList{t.buildSkippedTipsetReport(ts, name, start, reason)}
			ll.Infow("task skipped", "task", name, "reason", reason)
		}
		for name := range t.actorProcessors {
			taskOutputs[name] = model.PersistableList{t.buildSkippedTipsetReport(ts, name, start, reason)}
			ll.Infow("task skipped", "task", name, "reason", reason)
		}
	}

	// Wait for all tasks to complete
	completed := map[string]struct{}{}
	for inFlight > 0 {
		var res *TaskResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tctx.Done():
			// if the indexers timeout (window) context is done then we have run out of time.
			// loop over all tasks expected to complete, if they have not been completed mark them as skipped
			// then goto persistence routine.
			for _, name := range t.tasks {
				if _, complete := completed[name]; !complete {
					taskOutputs[name] = model.PersistableList{t.buildSkippedTipsetReport(ts, name, start, "indexer not ready")}
					ll.Infow("task skipped", "task", name, "reason", "indexer not ready")
					span.AddEvent(fmt.Sprintf("skipped task: %s", name))
				}
			}
			stats.Record(ctx, metrics.TipSetSkip.M(1))
			goto persist
		case res = <-results:
		}
		span.AddEvent(fmt.Sprintf("completed task: %s", res.Task))
		inFlight--

		llt := ll.With("task", res.Task)

		// Was there a fatal error?
		if res.Error != nil {
			llt.Errorw("task returned with error", "error", res.Error.Error())
			// tell all the processors to close their connections to the lens, they can reopen when needed
			return res.Error
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

			llt.Debugw("task report", "status", res.Report[idx].Status, "time", res.Report[idx].CompletedAt.Sub(res.Report[idx].StartedAt))
		}

		// Persist the processing report and the data in a single transaction
		taskOutputs[res.Task] = model.PersistableList{res.Report, res.Data}
		completed[res.Task] = struct{}{}
	}
	ll.Debugw("data extracted", "time", time.Since(start))

	if len(taskOutputs) == 0 {
		// Nothing to persist
		ll.Infow("tasks complete, nothing to persist", "total_time", time.Since(start))
		return nil
	}

persist:
	// wait until there is an empty slot before persisting
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.persistSlot <- struct{}{}:
		// Slot was free so we can continue. Slot is now taken.
	}

	// Persist all results
	go func() {
		ctx, persistSpan := otel.Tracer("").Start(ctx, "TipSetIndexer.Persist")
		if persistSpan.IsRecording() {
			persistSpan.SetAttributes(
				attribute.String("tipset", ts.String()),
				attribute.Int64("height", int64(ts.Height())),
			)
		}
		// free up the slot when done
		defer func() {
			persistSpan.End()
			<-t.persistSlot
		}()

		ll.Debugw("persisting data", "time", time.Since(start))
		var wg sync.WaitGroup
		wg.Add(len(taskOutputs))

		// Persist each processor's data concurrently since they don't overlap
		for task, p := range taskOutputs {
			go func(task string, p model.Persistable) {
				defer wg.Done()
				start := time.Now()
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
		ll.Infow("tasks complete", "total_time", time.Since(start))
	}()
	return nil
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

// getGenesisActors returns a map of all actors contained in the genesis block.
func (t *TipSetIndexer) getGenesisActors(ctx context.Context) (map[string]lens.ActorStateChange, error) {
	out := map[string]lens.ActorStateChange{}

	genesis, err := t.node.ChainGetGenesis(ctx)
	if err != nil {
		return nil, err
	}
	root, _, err := getStateTreeMapCIDAndVersion(ctx, t.node.Store(), genesis.ParentState())
	if err != nil {
		return nil, err
	}
	tree, err := state.LoadStateTree(t.node.Store(), root)
	if err != nil {
		return nil, err
	}
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		out[addr.String()] = lens.ActorStateChange{
			Actor:      *act,
			ChangeType: lens.ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// stateChangedActors is an optimized version of the lotus API method StateChangedActors. This method takes advantage of the efficient hamt/v3 diffing logic
// and applies it to versions of state tress supporting it. These include Version 2 and 3 of the lotus state tree implementation.
// stateChangedActors will fall back to the lotus API method when the optimized diffing cannot be applied.
func (t *TipSetIndexer) stateChangedActors(ctx context.Context, old, new cid.Cid) (map[string]lens.ActorStateChange, error) {
	ctx, span := otel.Tracer("").Start(ctx, "TipSetIndexer.StateChangedActors")
	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("old", old.String()),
			attribute.String("new", new.String()),
		)
	}
	defer span.End()

	var (
		buf = bytes.NewReader(nil)
		out = map[string]lens.ActorStateChange{}
	)

	oldRoot, oldVersion, err := getStateTreeMapCIDAndVersion(ctx, t.node.Store(), old)
	if err != nil {
		return nil, err
	}
	newRoot, newVersion, err := getStateTreeMapCIDAndVersion(ctx, t.node.Store(), new)
	if err != nil {
		return nil, err
	}

	if newVersion == oldVersion && (newVersion != types.StateTreeVersion0 && newVersion != types.StateTreeVersion1) {
		span.SetAttributes(attribute.String("diff", "fast"))
		// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
		changes, err := hamt.Diff(ctx, t.node.Store(), t.node.Store(), oldRoot, newRoot, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
			res := sha256.Sum256(input)
			return res[:]
		}))
		if err != nil {
			log.Errorw("failed to diff state tree efficiently, falling back to slow method", "error", err)
		} else {
			for _, change := range changes {
				addr, err := address.NewFromBytes([]byte(change.Key))
				if err != nil {
					return nil, xerrors.Errorf("address in state tree was not valid: %w", err)
				}
				var ch lens.ActorStateChange
				switch change.Type {
				case hamt.Add:
					ch.ChangeType = lens.ChangeTypeAdd
					buf.Reset(change.After.Raw)
					err = ch.Actor.UnmarshalCBOR(buf)
					buf.Reset(nil)
					if err != nil {
						return nil, err
					}
				case hamt.Remove:
					ch.ChangeType = lens.ChangeTypeRemove
					buf.Reset(change.Before.Raw)
					err = ch.Actor.UnmarshalCBOR(buf)
					buf.Reset(nil)
					if err != nil {
						return nil, err
					}
				case hamt.Modify:
					ch.ChangeType = lens.ChangeTypeModify
					buf.Reset(change.After.Raw)
					err = ch.Actor.UnmarshalCBOR(buf)
					buf.Reset(nil)
					if err != nil {
						return nil, err
					}
				}
				out[addr.String()] = ch
			}
			return out, nil
		}
	}
	span.SetAttributes(attribute.String("diff", "slow"))
	log.Debug("using slow state diff")
	actors, err := t.node.StateChangedActors(ctx, old, new)
	if err != nil {
		return nil, err
	}

	for addr, act := range actors {
		out[addr] = lens.ActorStateChange{
			Actor:      act,
			ChangeType: lens.ChangeTypeUnknown,
		}
	}

	return out, nil
}

func (t *TipSetIndexer) runMessageProcessor(ctx context.Context, p MessageProcessor, name string, ts, pts *types.TipSet, emsgs []*lens.ExecutedMessage, blkMsgs []*lens.BlockMessages, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.MessageProcessor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessMessages(ctx, ts, pts, emsgs, blkMsgs)
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

func (t *TipSetIndexer) runConsensusProcessor(ctx context.Context, p TipSetsProcessor, name string, ts, pts *types.TipSet, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.ConsensusProcessor.%s", name))
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
		Report:      report,
		Data:        data,
		StartedAt:   start,
		CompletedAt: time.Now(),
	}
}

func (t *TipSetIndexer) runActorProcessor(ctx context.Context, p ActorProcessor, name string, ts, pts *types.TipSet, actors map[string]lens.ActorStateChange, emsgs []*lens.ExecutedMessage, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.ActorProcessor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessActors(ctx, ts, pts, actors, emsgs)
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

func (t *TipSetIndexer) runMessageExecutionProcessor(ctx context.Context, p MessageExecutionProcessor, name string, ts, pts *types.TipSet, imsgs []*lens.MessageExecution, results chan *TaskResult) {
	ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("TipSetIndexer.MessageExecutionProcessor.%s", name))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, name))
	stats.Record(ctx, metrics.TipsetHeight.M(int64(ts.Height())))
	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()
	start := time.Now()

	data, report, err := p.ProcessMessageExecutions(ctx, t.node.Store(), ts, pts, imsgs)
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

// SkipTipSet writes a processing report to storage for each indexer task to indicate that the entire tipset
// was not processed.
func (t *TipSetIndexer) SkipTipSet(ctx context.Context, ts *types.TipSet, reason string) error {
	var reports model.PersistableList

	timestamp := time.Now()
	for name := range t.processors {
		reports = append(reports, t.buildSkippedTipsetReport(ts, name, timestamp, reason))
	}

	for name := range t.messageProcessors {
		reports = append(reports, t.buildSkippedTipsetReport(ts, name, timestamp, reason))
	}

	for name := range t.messageExecutionProcessors {
		reports = append(reports, t.buildSkippedTipsetReport(ts, name, timestamp, reason))
	}

	for name := range t.actorProcessors {
		reports = append(reports, t.buildSkippedTipsetReport(ts, name, timestamp, reason))
	}

	for name := range t.consensusProcessor {
		reports = append(reports, t.buildSkippedTipsetReport(ts, name, timestamp, reason))
	}

	if err := t.storage.PersistBatch(ctx, reports...); err != nil {
		return xerrors.Errorf("persist reports: %w", err)
	}
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
	ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.Persistable, *visormodel.ProcessingReport, error)
}

type TipSetsProcessor interface {
	// ProcessTipSets processes a parent and child tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSets(ctx context.Context, child, parent *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error)
}

type MessageProcessor interface {
	// ProcessMessages processes messages contained within a tipset. If error is non-nil then the processor encountered a fatal error.
	// pts is the tipset containing the messages, ts is the tipset containing the receipts
	// Any data returned must be accompanied by a processing report.
	ProcessMessages(ctx context.Context, ts *types.TipSet, pts *types.TipSet, emsgs []*lens.ExecutedMessage, blkMsgs []*lens.BlockMessages) (model.Persistable, *visormodel.ProcessingReport, error)
}

type MessageExecutionProcessor interface {
	ProcessMessageExecutions(ctx context.Context, store adt.Store, ts *types.TipSet, pts *types.TipSet, imsgs []*lens.MessageExecution) (model.Persistable, *visormodel.ProcessingReport, error)
}

type ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessActors(ctx context.Context, ts *types.TipSet, pts *types.TipSet, actors map[string]lens.ActorStateChange, emsgs []*lens.ExecutedMessage) (model.Persistable, *visormodel.ProcessingReport, error)
}
