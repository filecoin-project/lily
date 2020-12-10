package chain

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/indexer"
)

const (
	ActorStateTask       = "actorstates"       // task that extracts both raw and parsed actor states
	ActorRawStateTask    = "actorstatesraw"    // task that only extracts raw actor state
	ActorParsedStateTask = "actorstatesparsed" // task that only extracts parsed actor state
	BlocksTask           = "blocks"            // task that extracts block data
	MessagesTask         = "messages"          // task that extracts message data
	ChainEconomicsTask   = "chaineconomics"    // task that extracts chain economics data
)

var log = logging.Logger("chain")

var _ indexer.TipSetObserver = (*TipSetIndexer)(nil)

// A TipSetWatcher waits for tipsets and persists their block data into a database.
type TipSetIndexer struct {
	window      time.Duration
	storage     Storage
	processors  map[string]TipSetProcessor
	name        string
	persistSlot chan struct{}
}

// A TipSetIndexer extracts block, message and actor state data from a tipset and persists it to storage. Extraction
// and persistence are concurrent. Extraction of the a tipset can proceed while data from the previous extraction is
// being persisted. The indexer may be given a time window in which to complete data extraction. The name of the
// indexer is used as the reporter in the visor_processing_reports table.
func NewTipSetIndexer(o lens.APIOpener, d Storage, window time.Duration, name string, tasks []string) (*TipSetIndexer, error) {
	tsi := &TipSetIndexer{
		storage:     d,
		window:      window,
		name:        name,
		persistSlot: make(chan struct{}, 1), // allow one concurrent persistence job
		processors:  map[string]TipSetProcessor{},
	}

	for _, task := range tasks {
		switch task {
		case BlocksTask:
			tsi.processors[BlocksTask] = NewBlockProcessor()
		case MessagesTask:
			tsi.processors[MessagesTask] = NewMessageProcessor(o)
		case ChainEconomicsTask:
			tsi.processors[ChainEconomicsTask] = NewChainEconomicsProcessor(o)
		case ActorStateTask:
			tsi.processors[ActorStateTask] = NewActorStateProcessor(o, true, true)
		case ActorRawStateTask:
			tsi.processors[ActorRawStateTask] = NewActorStateProcessor(o, true, false)
		case ActorParsedStateTask:
			tsi.processors[ActorRawStateTask] = NewActorStateProcessor(o, false, true)
		default:
			return nil, xerrors.Errorf("unknown task: %s", task)
		}
	}
	return tsi, nil
}

// TipSet is called when a new tipset has been discovered
func (t *TipSetIndexer) TipSet(ctx context.Context, ts *types.TipSet) error {
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

	// Run each task concurrently
	results := make(chan *TaskResult, len(t.processors))
	for name, p := range t.processors {
		go t.runProcessor(tctx, p, name, ts, results)
	}

	ll := log.With("height", int64(ts.Height()))

	// A map to gather the persistable outputs from each task
	taskOutputs := make(map[string]PersistableWithTxList, len(t.processors))

	// Wait for all tasks to complete
	inFlight := len(t.processors)
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
		taskOutputs[res.Task] = PersistableWithTxList{res.Report, res.Data}
	}

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
			go func(task string, p model.PersistableWithTx) {
				defer wg.Done()
				if err := t.storage.Persist(ctx, p); err != nil {
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
	data, report, err := p.ProcessTipSet(ctx, ts)
	if err != nil {
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
	for name, p := range t.processors {
		if err := p.Close(); err != nil {
			log.Errorw("error received while closing task processor", "error", err, "task", name)
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
	Data   model.PersistableWithTx
}

type TipSetProcessor interface {
	// ProcessTipSet processes a tipset. If error is non-nil then the processor encountered a fatal error.
	// Any data returned must be accompanied by a processing report.
	ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.PersistableWithTx, *visormodel.ProcessingReport, error)
	Close() error
}
