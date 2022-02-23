package lily

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/lens/lily/modules"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/impl/net"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/network"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/lily/storage"
)

var _ LilyAPI = (*LilyNodeAPI)(nil)

type LilyNodeAPI struct {
	fx.In `ignore-unexported:"true"`

	net.NetAPI
	full.ChainAPI
	full.StateAPI
	full.SyncAPI
	common.CommonAPI
	Events         *events.Events
	Scheduler      *schedule.Scheduler
	StorageCatalog *storage.Catalog
	ExecMonitor    stmgr.ExecMonitor
	CacheConfig    *util.CacheConfig
	actorStore     adt.Store
	actorStoreInit sync.Once
}

func (m *LilyNodeAPI) ChainGetTipSetAfterHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	// TODO (Frrist): I copied this from lotus, I need it now to handle gap filling edge cases.
	ts, err := m.ChainAPI.Chain.GetTipSetFromKey(key)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", key, err)
	}
	return m.ChainAPI.Chain.GetTipsetByHeight(ctx, epoch, ts, false)
}

func (m *LilyNodeAPI) Daemonized() bool {
	return true
}

func (m *LilyNodeAPI) LilyWatch(_ context.Context, cfg *LilyWatchConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.Storage, md)
	if err != nil {
		return nil, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	indexer, err := chain.NewTipSetIndexer(m, strg, cfg.Window, cfg.Name, cfg.Tasks)
	if err != nil {
		return nil, err
	}

	// HeadNotifier bridges between the event system and the watcher
	obs := &HeadNotifier{
		bufferSize: 5,
	}

	// Hook up the notifier to the event system
	head := m.Events.Observe(obs)
	if err := obs.SetCurrent(ctx, head); err != nil {
		return nil, err
	}

	// warm the tipset cache.
	tsCache := chain.NewTipSetCache(cfg.Confidence)
	if err := tsCache.Warm(ctx, head, m.ChainModuleAPI.ChainGetTipSet); err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.Name,
		Type: "watch",
		Params: map[string]string{
			"window":     cfg.Window.String(),
			"confidence": fmt.Sprintf("%d", cfg.Confidence),
			"storage":    cfg.Storage,
		},
		Tasks:               cfg.Tasks,
		Job:                 chain.NewWatcher(indexer, obs, tsCache),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyWalk(_ context.Context, cfg *LilyWalkConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.Storage, md)
	if err != nil {
		return nil, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	indexer, err := chain.NewTipSetIndexer(m, strg, cfg.Window, cfg.Name, cfg.Tasks)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.Name,
		Type: "walk",
		Params: map[string]string{
			"window":    cfg.Window.String(),
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.Storage,
		},
		Tasks:               cfg.Tasks,
		Job:                 chain.NewWalker(indexer, m, cfg.From, cfg.To),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyGapFind(_ context.Context, cfg *LilyGapFindConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.Storage, md)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name:  cfg.Name,
		Type:  "Find",
		Tasks: cfg.Tasks,
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.Storage,
		},
		Job:                 chain.NewGapIndexer(m, db, cfg.Name, cfg.From, cfg.To, cfg.Tasks),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyGapFill(_ context.Context, cfg *LilyGapFillConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.Storage, md)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.Name,
		Type: "Fill",
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.Storage,
		},
		Tasks:               cfg.Tasks,
		Job:                 chain.NewGapFiller(m, db, cfg.Name, cfg.From, cfg.To, cfg.Tasks),
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return res, nil
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

func (m *LilyNodeAPI) LilyJobWait(ctx context.Context, ID schedule.JobID) (*schedule.JobListResult, error) {
	res, err := m.Scheduler.WaitJob(ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *LilyNodeAPI) LilyJobList(_ context.Context) ([]schedule.JobListResult, error) {
	return m.Scheduler.Jobs(), nil
}

func (m *LilyNodeAPI) GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
	return util.GetExecutedAndBlockMessagesForTipset(ctx, m.ChainAPI.Chain, m.StateManager, ts, pts)
}

func (m *LilyNodeAPI) GetMessageExecutionsForTipSet(ctx context.Context, next *types.TipSet, current *types.TipSet) ([]*lens.MessageExecution, error) {
	// this is defined in the lily daemon dep injection constructor, failure here is a developer error.
	msgMonitor, ok := m.ExecMonitor.(*modules.BufferedExecMonitor)
	if !ok {
		panic(fmt.Sprintf("bad cast, developer error expected modules.BufferedExecMonitor, got %T", m.ExecMonitor))
	}

	// if lily was watching the chain when this tipset was applied then its exec monitor will already
	// contain executions for this tipset.
	executions, err := msgMonitor.ExecutionFor(current)
	if err != nil {
		if err == modules.ExecutionTraceNotFound {
			// if lily hasn't watched this tipset be applied then we need to compute its execution trace.
			// this will likely be the case for most walk tasks.
			_, err := m.StateManager.ExecutionTraceWithMonitor(ctx, current, msgMonitor)
			if err != nil {
				return nil, xerrors.Errorf("failed to compute execution trace for tipset %s: %w", current.Key().String(), err)
			}
			// the above call will populate the msgMonitor with an execution trace for this tipset, get it.
			executions, err = msgMonitor.ExecutionFor(current)
			if err != nil {
				return nil, xerrors.Errorf("failed to find execution trace for tipset %s: %w", current.Key().String(), err)
			}
		} else {
			return nil, xerrors.Errorf("failed to extract message execution for tipset %s: %w", next, err)
		}
	}

	getActorCode, err := util.MakeGetActorCodeFunc(ctx, m.ChainAPI.Chain.ActorStore(ctx), next, current)
	if err != nil {
		return nil, xerrors.Errorf("failed to make actor code query function: %w", err)
	}

	out := make([]*lens.MessageExecution, len(executions))
	for idx, execution := range executions {
		toCode, found := getActorCode(execution.Msg.To)
		// if the message failed to execute due to lack of gas then the TO actor may never have been created.
		if !found {
			log.Warnw("failed to find TO actor", "height", next.Height().String(), "message", execution.Msg.Cid().String(), "actor", execution.Msg.To.String())
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
	m.actorStoreInit.Do(func() {
		if m.CacheConfig.StatestoreCacheSize > 0 {
			var err error
			log.Infof("creating caching statestore with size=%d", m.CacheConfig.StatestoreCacheSize)
			m.actorStore, err = util.NewCachingStateStore(m.ChainAPI.Chain.StateBlockstore(), int(m.CacheConfig.StatestoreCacheSize))
			if err == nil {
				return // done
			}

			log.Errorf("failed to create caching statestore: %v", err)
		}

		m.actorStore = m.ChainAPI.Chain.ActorStore(context.TODO())
	})

	return m.actorStore
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

func (m *LilyNodeAPI) LogSetLevelRegex(ctx context.Context, regex, level string) error {
	return logging.SetLogLevelRegex(regex, level)
}

func (m *LilyNodeAPI) Shutdown(ctx context.Context) error {
	m.ShutdownChan <- struct{}{}
	return nil
}

func (m *LilyNodeAPI) LilySurvey(_ context.Context, cfg *LilySurveyConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.Storage, storage.Metadata{JobName: cfg.Name})
	if err != nil {
		return nil, err
	}

	// instantiate a new surveyer.
	surv, err := network.NewSurveyer(m, strg, cfg.Interval, cfg.Name, cfg.Tasks)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name:  cfg.Name,
		Tasks: cfg.Tasks,
		Job:   surv,
		Params: map[string]string{
			"interval": cfg.Interval.String(),
		},
		RestartOnFailure:    cfg.RestartOnFailure,
		RestartOnCompletion: cfg.RestartOnCompletion,
		RestartDelay:        cfg.RestartDelay,
	})

	return res, nil
}

var _ events.TipSetObserver = (*HeadNotifier)(nil)

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

	log.Debugw("head notifier setting head", "tipset", ts.Key().String())
	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventCurrent,
		TipSet: ts,
	}
	return nil
}

func (h *HeadNotifier) Apply(ctx context.Context, from, to *types.TipSet) error {
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

	log.Debugw("head notifier apply", "to", to.Key().String(), "from", from.Key().String())
	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventApply,
		TipSet: to,
	}
	return nil
}

func (h *HeadNotifier) Revert(ctx context.Context, from, to *types.TipSet) error {
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

	log.Debugw("head notifier revert", "to", to.Key().String(), "from", from.Key().String())
	ev <- &chain.HeadEvent{
		Type:   chain.HeadEventRevert,
		TipSet: from,
	}
	return nil
}

// used for debugging querries, call ORM.AddHook and this will print all queries.
type LogQueryHook struct{}

func (l *LogQueryHook) BeforeQuery(ctx context.Context, evt *pg.QueryEvent) (context.Context, error) {
	q, err := evt.FormattedQuery()
	if err != nil {
		return nil, err
	}

	if evt.Err != nil {
		fmt.Printf("%s executing a query:\n%s\n", evt.Err, q)
	}

	fmt.Println(string(q))

	return ctx, nil
}

func (l *LogQueryHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	return nil
}
