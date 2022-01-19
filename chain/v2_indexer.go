package chain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"
	"sync"
	"time"
)

type V2TipSetsProcessor interface {
	// ProcessTipSets processes a parent and child tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSets(ctx context.Context, current, previous *types.TipSet) (model.Persistable, visormodel.ProcessingReportList, error)
}

type V2ActorProcessor interface {
	// ProcessActors processes a set of actors. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessActors(ctx context.Context, current *types.TipSet, previous *types.TipSet, actors map[string]lens.ActorStateChange) (model.Persistable, *visormodel.ProcessingReport, error)
}
type ActorExtractorProcessor interface {
	ProcessActorExtractors(ctx context.Context, current *types.TipSet, previous *types.TipSet, actors map[address.Address]lens.ActorStateChange, extractors []ActorStateExtractor) (model.Persistable, *visormodel.ProcessingReport, error)
}

type V2TipSetIndexer struct {
	window          time.Duration
	storage         model.Storage
	processors      map[string]V2TipSetsProcessor
	actorProcessors map[string]V2ActorProcessor
	name            string
	persistSlot     chan struct{} // filled with a token when a goroutine is persisting data
	lastTipSet      *types.TipSet
	node            lens.API
	tasks           []string

	tsExtractors  []TipSetStateExtractor
	actExtractors []ActorStateExtractor
}

// Passes TestLilyVectorWalkExtraction
func NewV2TipSetIndexer(node lens.API, d model.Storage, name string, tasks []string) (*V2TipSetIndexer, error) {
	tsi := &V2TipSetIndexer{
		storage:         d,
		name:            name,
		persistSlot:     make(chan struct{}, 1), // allow one concurrent persistence job
		processors:      map[string]V2TipSetsProcessor{},
		actorProcessors: map[string]V2ActorProcessor{},
		node:            node,
		tasks:           tasks,
	}

	for _, modelName := range []string{"actors"} {
		extractableModel, exType, err := StringToModelTypeAndExtractor(modelName)
		if err != nil {
			return nil, err
		}

		switch exType {
		case TipSetStateExtractorType:
			ex := TipSetExtractorForModel(extractableModel)
			tsi.tsExtractors = append(tsi.tsExtractors, ex)
		case ActorStateExtractorType:
			ex := ActorStateExtractorForModel(extractableModel)
			tsi.actExtractors = append(tsi.actExtractors, ex)
		}
	}

	return tsi, nil
}

// - Window was removed. Callers of this are responsible for setting the window.
func (t *V2TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
	pts, err := t.node.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return err
	}

	var (
		current  = ts
		previous = pts
		results  = make(chan *TaskResult, len(t.processors)+len(t.actorProcessors))
	)

	start := time.Now()
	log.Infow("processing", "current", current.Height(), "previous", previous.Height())

	if err := t.index(ctx, current, previous, results); err != nil {
		return err
	}

	log.Infow("indexed", "current", current.Height(), "previous", previous.Height(), "duration", time.Since(start))

	if err := t.persist(ctx, results); err != nil {
		return err
	}

	log.Infow("persisted", "current", current.Height(), "previous", previous.Height(), "duration", time.Since(start))

	return nil
}

func (t *V2TipSetIndexer) Close() error {
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

func (t *V2TipSetIndexer) index(ctx context.Context, current, previous *types.TipSet, results chan *TaskResult) error {
	if len(t.actExtractors) > 0 {
		var (
			changes map[address.Address]lens.ActorStateChange
			err     error
		)
		// the diff between the first tipset mined and genesis is nil, this is becasue the actors haven't changed, but they
		// _do_ have state, we list it here instead of diff it.
		if current.Height() == 1 {
			changes, err = t.getGenesisActors(ctx)
		} else {
			changes, err = t.stateChangedActors(ctx, current.ParentState(), previous.ParentState())
			if err != nil {
				return err
			}
		}

		actorProcessor := NewActorExtractorProcessorImpl(t.node)
		go t.runActorExtractorProcessor(ctx, actorProcessor, current, previous, changes, results)
	}

	return nil
}

func (t *V2TipSetIndexer) persist(ctx context.Context, results chan *TaskResult) error {
	inFlightTasks := len(t.actExtractors)
	taskOutputs := make(map[string]model.PersistableList, inFlightTasks)
	// Wait for all tasks to complete
	for inFlightTasks > 0 {
		var res *TaskResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res = <-results:
		}
		inFlightTasks--

		// Was there a fatal error?
		if res.Error != nil {
			return res.Error
		}

		if res.Report == nil || len(res.Report) == 0 {
			// Nothing was done for this tipset
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
		}

		// Persist the processing report and the data in a single transaction
		taskOutputs[res.Task] = model.PersistableList{res.Report, res.Data}
	}

	if len(taskOutputs) == 0 {
		// Nothing to persist
		return nil
	}

	// wait until there is an empty slot before persisting
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.persistSlot <- struct{}{}:
		// Slot was free so we can continue. Slot is now taken.
	}

	// Persist all results
	go func() {
		// free up the slot when done
		defer func() {
			<-t.persistSlot
		}()

		var wg sync.WaitGroup
		wg.Add(len(taskOutputs))

		// Persist each processor's data concurrently since they don't overlap
		for task, p := range taskOutputs {
			go func(task string, p model.Persistable) {
				defer wg.Done()
				ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, task))

				if err := t.storage.PersistBatch(ctx, p); err != nil {
					stats.Record(ctx, metrics.PersistFailure.M(1))
					return
				}
			}(task, p)
		}
		wg.Wait()
	}()
	return nil
}

func (t *V2TipSetIndexer) runProcessors(ctx context.Context, p V2TipSetsProcessor, name string, current, previous *types.TipSet, results chan *TaskResult) {
	start := time.Now()

	data, report, err := p.ProcessTipSets(ctx, current, previous)
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

func (t *V2TipSetIndexer) runActorExtractorProcessor(ctx context.Context, p ActorExtractorProcessor, current, previous *types.TipSet, actors map[address.Address]lens.ActorStateChange, results chan *TaskResult) {
	start := time.Now()

	data, report, err := p.ProcessActorExtractors(ctx, current, previous, actors, t.actExtractors)
	if err != nil {
		stats.Record(ctx, metrics.ProcessingFailure.M(1))
		results <- &TaskResult{
			Task:        "thing", // TODO
			Error:       err,
			StartedAt:   start,
			CompletedAt: time.Now(),
		}
		return
	}
	results <- &TaskResult{
		Task:        "thing", // TODO
		Report:      visormodel.ProcessingReportList{report},
		Data:        data,
		StartedAt:   start,
		CompletedAt: time.Now(),
	}
}

func (t *V2TipSetIndexer) runActorProcessor(ctx context.Context, p V2ActorProcessor, name string, current, previous *types.TipSet, actors map[string]lens.ActorStateChange, results chan *TaskResult) {
	start := time.Now()

	data, report, err := p.ProcessActors(ctx, current, previous, actors)
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

// stateChangedActors is an optimized version of the lotus API method StateChangedActors. This method takes advantage of the efficient hamt/v3 diffing logic
// and applies it to versions of state tress supporting it. These include Version 2 and 3 of the lotus state tree implementation.
// stateChangedActors will fall back to the lotus API method when the optimized diffing cannot be applied.
func (t *V2TipSetIndexer) stateChangedActors(ctx context.Context, current, previous cid.Cid) (map[address.Address]lens.ActorStateChange, error) {
	var (
		buf = bytes.NewReader(nil)
		out = map[address.Address]lens.ActorStateChange{}
	)

	previousRood, previousVersion, err := getStateTreeMapCIDAndVersion(ctx, t.node.Store(), previous)
	if err != nil {
		return nil, err
	}
	currentRoot, currentVersion, err := getStateTreeMapCIDAndVersion(ctx, t.node.Store(), current)
	if err != nil {
		return nil, err
	}

	if currentVersion == previousVersion && (currentVersion != types.StateTreeVersion0 && currentVersion != types.StateTreeVersion1) {
		// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
		changes, err := hamt.Diff(ctx, t.node.Store(), t.node.Store(), previousRood, currentRoot,
			hamt.UseTreeBitWidth(5),
			hamt.UseHashFunction(func(input []byte) []byte {
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
				out[addr] = ch
			}
			return out, nil
		}
	}
	log.Debug("using slow state diff")
	actors, err := t.node.StateChangedActors(ctx, previous, current)
	if err != nil {
		return nil, err
	}

	for addrStr, act := range actors {
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return nil, err
		}
		out[addr] = lens.ActorStateChange{
			Actor:      act,
			ChangeType: lens.ChangeTypeUnknown,
		}
	}

	return out, nil
}

// getGenesisActors returns a map of all actors contained in the genesis block.
func (t *V2TipSetIndexer) getGenesisActors(ctx context.Context) (map[address.Address]lens.ActorStateChange, error) {
	out := map[address.Address]lens.ActorStateChange{}

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
		out[addr] = lens.ActorStateChange{
			Actor:      *act,
			ChangeType: lens.ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}
