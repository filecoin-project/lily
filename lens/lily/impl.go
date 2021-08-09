package lily

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/sentinel-visor/lens/lily/modules"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var _ LilyAPI = (*LilyNodeAPI)(nil)

type LilyNodeAPI struct {
	fx.In

	full.ChainAPI
	full.StateAPI
	full.SyncAPI
	common.CommonAPI
	Events         *events.Events
	Scheduler      *schedule.Scheduler
	StorageCatalog *storage.Catalog
	ExecMonitor    stmgr.ExecMonitor
}

func (m *LilyNodeAPI) Daemonized() bool {
	return true
}

func (m *LilyNodeAPI) LilyWatch(_ context.Context, cfg *LilyWatchConfig) (schedule.JobID, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.Storage, md)
	if err != nil {
		return schedule.InvalidJobID, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	indexer, err := chain.NewTipSetIndexer(m, strg, cfg.Window, cfg.Name, cfg.Tasks)
	if err != nil {
		return schedule.InvalidJobID, err
	}

	// HeadNotifier bridges between the event system and the watcher
	obs := &HeadNotifier{
		bufferSize: 5,
	}

	// get the current head and set it on the tipset cache (mimic chain.watcher behaviour)
	head, err := m.ChainModuleAPI.ChainHead(ctx)
	if err != nil {
		return schedule.InvalidJobID, err
	}

	// Won't block since we are using non-zero buffer size in head notifier
	if err := obs.SetCurrent(ctx, head); err != nil {
		log.Errorw("failed to set current head tipset", "error", err)
	}

	// Hook up the notifier to the event system
	if err := m.Events.Observe(obs); err != nil {
		return schedule.InvalidJobID, err
	}

	id := m.Scheduler.Submit(&schedule.JobConfig{
		Name:                cfg.Name,
		Tasks:               cfg.Tasks,
		Job:                 chain.NewWatcher(indexer, obs, cfg.Confidence),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return id, nil
}

func (m *LilyNodeAPI) LilyWalk(_ context.Context, cfg *LilyWalkConfig) (schedule.JobID, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.Storage, md)
	if err != nil {
		return schedule.InvalidJobID, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	indexer, err := chain.NewTipSetIndexer(m, strg, cfg.Window, cfg.Name, cfg.Tasks)
	if err != nil {
		return schedule.InvalidJobID, err
	}

	id := m.Scheduler.Submit(&schedule.JobConfig{
		Name:                cfg.Name,
		Tasks:               cfg.Tasks,
		Job:                 chain.NewWalker(indexer, m, cfg.From, cfg.To),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return id, nil
}

func (m *LilyNodeAPI) LilyJobStart(_ context.Context, ID schedule.JobID) error {
	if err := m.Scheduler.StartJob(ID); err != nil {
		return err
	}
	return nil
}

func (m *LilyNodeAPI) LilyJobStop(_ context.Context, ID schedule.JobID) error {
	if err := m.Scheduler.StopJob(ID); err != nil {
		return err
	}
	return nil
}

func (m *LilyNodeAPI) LilyJobList(_ context.Context) ([]schedule.JobResult, error) {
	return m.Scheduler.Jobs(), nil
}

func (m *LilyNodeAPI) Open(_ context.Context) (lens.API, lens.APICloser, error) {
	return m, func() {}, nil
}

func (m *LilyNodeAPI) GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
	return util.GetExecutedAndBlockMessagesForTipset(ctx, m.ChainAPI.Chain, ts, pts)
}

func (m *LilyNodeAPI) GetMessageExecutionsForTipSet(ctx context.Context, ts *types.TipSet, pts *types.TipSet) ([]*lens.MessageExecution, error) {
	// this is defined in the lily daemon dep injection constructor, failure here is a developer error.
	msgMonitor, ok := m.ExecMonitor.(*modules.BufferedExecMonitor)
	if !ok {
		panic(fmt.Sprintf("bad cast, developer error expected modules.BufferedExecMonitor, got %T", m.ExecMonitor))
	}

	// if lily was watching the chain when this tipset was applied then its exec monitor will already
	// contain executions for this tipset.
	executions, err := msgMonitor.ExecutionFor(pts) //lint:ignore SA4006 false positive
	if err == modules.ExecutionTraceNotFound {
		// if lily hasn't watched this tipset be applied then we need to compute its execution trace.
		// this will likely be the case for most walk tasks.
		_, err := m.StateManager.ExecutionTraceWithMonitor(ctx, pts, msgMonitor)
		if err != nil {
			return nil, xerrors.Errorf("failed to compute execution trace for tipset: %s", pts.Key().String())
		}
		// the above call will populate the msgMonitor with an execution trace for this tipset, get it.
		executions, err = msgMonitor.ExecutionFor(pts)
		if err != nil {
			return nil, xerrors.Errorf("failed to find execution trace for tipset: %s", pts.Key().String())
		}
	} else {
		return nil, xerrors.Errorf("failed to extract message execution for tipset %s: %w", ts, err)
	}

	getActorCode, err := util.MakeGetActorCodeFunc(ctx, m.ChainAPI.Chain.ActorStore(ctx), ts, pts)
	if err != nil {
		return nil, xerrors.Errorf("failed to make actor code query function: %w", err)
	}

	out := make([]*lens.MessageExecution, len(executions))
	for idx, execution := range executions {
		toCode, found := getActorCode(execution.Msg.To)
		// if the message failed to execute due to lack of gas then the TO actor may never have been created.
		if !found {
			log.Warnw("failed to find TO actor", "height", ts.Height().String(), "message", execution.Msg.Cid().String(), "actor", execution.Msg.To.String())
		}
		// if the message sender cannot be found this is an unexpected error
		fromCode, found := getActorCode(execution.Msg.From)
		if !found {
			return nil, xerrors.Errorf("failed to find from actor %s height %d message %s", execution.Msg.From, execution.TipSet.Height(), execution.Msg.Cid())
		}
		out[idx] = &lens.MessageExecution{
			Cid:           execution.Mcid,
			StateRoot:     execution.TipSet.ParentState(),
			Height:        execution.TipSet.Height(),
			Message:       execution.Msg,
			Ret:           execution.Ret,
			Implicit:      execution.Implicit,
			ToActorCode:   toCode,
			FromActorCode: fromCode,
		}
	}
	return out, nil
}

func (m *LilyNodeAPI) Store() adt.Store {
	return m.ChainAPI.Chain.ActorStore(context.TODO())
}

func (m *LilyNodeAPI) StateGetReceipt(ctx context.Context, msg cid.Cid, from types.TipSetKey) (*types.MessageReceipt, error) {
	ml, err := m.StateSearchMsg(ctx, from, msg, api.LookbackNoLimit, true)
	if err != nil {
		return nil, err
	}

	if ml == nil {
		return nil, nil
	}

	return &ml.Receipt, nil
}

func (m *LilyNodeAPI) LogList(ctx context.Context) ([]string, error) {
	return logging.GetSubsystems(), nil
}

func (m *LilyNodeAPI) LogSetLevel(ctx context.Context, subsystem, level string) error {
	return logging.SetLogLevel(subsystem, level)
}

func (m *LilyNodeAPI) Shutdown(ctx context.Context) error {
	m.ShutdownChan <- struct{}{}
	return nil
}

type HeadNotifier struct {
	mu     sync.Mutex            // protects following fields
	events chan *chain.HeadEvent // created lazily, closed by first cancel call
	err    error                 // set to non-nil by the first cancel call

	// size of the buffer to maintain for events. Using a buffer reduces chance
	// that the emitter of events will block when sending to this notifier.
	bufferSize int
}

func (h *HeadNotifier) eventsCh() chan *chain.HeadEvent {
	// caller must hold mu
	if h.events == nil {
		h.events = make(chan *chain.HeadEvent, h.bufferSize)
	}
	return h.events
}

func (h *HeadNotifier) HeadEvents() <-chan *chain.HeadEvent {
	h.mu.Lock()
	ev := h.eventsCh()
	h.mu.Unlock()
	return ev
}

func (h *HeadNotifier) Err() error {
	h.mu.Lock()
	err := h.err
	h.mu.Unlock()
	return err
}

func (h *HeadNotifier) Cancel(err error) {
	h.mu.Lock()
	if h.err != nil {
		h.mu.Unlock()
		return
	}
	h.err = err
	if h.events == nil {
		h.events = make(chan *chain.HeadEvent, h.bufferSize)
	}
	close(h.events)
	h.mu.Unlock()
}

func (h *HeadNotifier) SetCurrent(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventCurrent,
		TipSet: ts,
	}
	return nil
}

func (h *HeadNotifier) Apply(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventApply,
		TipSet: ts,
	}
	return nil
}

func (h *HeadNotifier) Revert(ctx context.Context, ts *types.TipSet) error {
	h.mu.Lock()
	if h.err != nil {
		err := h.err
		h.mu.Unlock()
		return err
	}
	ev := h.eventsCh()
	h.mu.Unlock()

	// This is imprecise since it's inherently racy but good enough to emit
	// a warning that the event may block the sender
	if len(ev) == cap(ev) {
		log.Warnw("head notifier buffer at capacity", "queued", len(ev))
	}

	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventRevert,
		TipSet: ts,
	}
	return nil
}
