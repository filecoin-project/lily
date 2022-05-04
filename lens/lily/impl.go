package lily

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
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

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/gap"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue"
	"github.com/filecoin-project/lily/chain/indexer/integrated"
	"github.com/filecoin-project/lily/chain/walk"
	"github.com/filecoin-project/lily/chain/watch"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/lily/modules"
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
	Events    *events.Events
	Scheduler *schedule.Scheduler

	ExecMonitor stmgr.ExecMonitor
	CacheConfig *util.CacheConfig

	StorageCatalog *storage.Catalog
	QueueCatalog   *distributed.Catalog

	actorStore     adt.Store
	actorStoreInit sync.Once
}

func (m *LilyNodeAPI) CirculatingSupply(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	return m.StateAPI.StateVMCirculatingSupplyInternal(ctx, key)
}

func (m *LilyNodeAPI) ChainGetTipSetAfterHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	// TODO (Frrist): I copied this from lotus, I need it now to handle gap filling edge cases.
	ts, err := m.ChainAPI.Chain.GetTipSetFromKey(ctx, key)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", key, err)
	}
	return m.ChainAPI.Chain.GetTipsetByHeight(ctx, epoch, ts, false)
}

func (m *LilyNodeAPI) StartTipSetWorker(_ context.Context, cfg *LilyTipSetWorkerConfig) (*schedule.JobSubmitResult, error) {
	ctx := context.Background()
	log.Infow("starting TipSetWorker", "name", cfg.JobConfig.Name)
	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	qcfg, err := m.QueueCatalog.AsynqConfig(cfg.Queue)
	if err != nil {
		return nil, err
	}

	taskAPI, err := datasource.NewDataSource(m)
	if err != nil {
		return nil, err
	}

	im, err := integrated.NewManager(taskAPI, strg, cfg.JobConfig.Name)
	if err != nil {
		return nil, err
	}

	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "tipset-worker",
		Params: map[string]string{
			"queue":       cfg.Queue,
			"storage":     cfg.JobConfig.Storage,
			"concurrency": strconv.Itoa(cfg.Concurrency),
		},
		Job:                 queue.NewAsynqWorker(im, db, cfg.JobConfig.Name, 1, qcfg),
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})
	return res, nil
}

func (m *LilyNodeAPI) LilyIndex(_ context.Context, cfg *LilyIndexConfig) (interface{}, error) {
	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	taskAPI, err := datasource.NewDataSource(m)
	if err != nil {
		return nil, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	im, err := integrated.NewManager(taskAPI, strg, cfg.JobConfig.Name, integrated.WithWindow(cfg.JobConfig.Window))
	if err != nil {
		return nil, err
	}

	ts, err := m.ChainGetTipSet(ctx, cfg.TipSet)
	if err != nil {
		return nil, err
	}

	success, err := im.TipSet(ctx, ts, indexer.WithTasks(cfg.JobConfig.Tasks))

	return success, err
}

func (m *LilyNodeAPI) LilyIndexNotify(_ context.Context, cfg *LilyIndexNotifyConfig) (interface{}, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	qcfg, err := m.QueueCatalog.AsynqConfig(cfg.Queue)
	if err != nil {
		return nil, err
	}

	ts, err := m.ChainGetTipSet(ctx, cfg.IndexConfig.TipSet)
	if err != nil {
		return nil, err
	}

	idx := distributed.NewTipSetIndexer(queue.NewAsynq(qcfg))

	return idx.TipSet(ctx, ts, indexer.WithIndexerType(indexer.Index), indexer.WithTasks(cfg.IndexConfig.JobConfig.Tasks))
}

type watcherAPIWrapper struct {
	*events.Events
	full.ChainModuleAPI
}

func (m *LilyNodeAPI) LilyWatch(_ context.Context, cfg *LilyWatchConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}

	wapi := &watcherAPIWrapper{
		Events:         m.Events,
		ChainModuleAPI: m.ChainModuleAPI,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	taskAPI, err := datasource.NewDataSource(m)
	if err != nil {
		return nil, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	idx, err := integrated.NewManager(taskAPI, strg, cfg.JobConfig.Name, integrated.WithWindow(cfg.JobConfig.Window))
	if err != nil {
		return nil, err
	}

	watchJob := watch.NewWatcher(wapi, idx, cfg.JobConfig.Name,
		watch.WithTasks(cfg.JobConfig.Tasks...),
		watch.WithConfidence(cfg.Confidence),
		watch.WithConcurrentWorkers(cfg.Workers),
		watch.WithBufferSize(cfg.BufferSize),
	)

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "watch",
		Params: map[string]string{
			"window":     cfg.JobConfig.Window.String(),
			"storage":    cfg.JobConfig.Storage,
			"confidence": strconv.Itoa(cfg.Confidence),
			"worker":     strconv.Itoa(cfg.Workers),
			"buffer":     strconv.Itoa(cfg.BufferSize),
		},
		Tasks:               cfg.JobConfig.Tasks,
		Job:                 watchJob,
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyWatchNotify(_ context.Context, cfg *LilyWatchNotifyConfig) (*schedule.JobSubmitResult, error) {
	wapi := &watcherAPIWrapper{
		Events:         m.Events,
		ChainModuleAPI: m.ChainModuleAPI,
	}

	qcfg, err := m.QueueCatalog.AsynqConfig(cfg.Queue)
	if err != nil {
		return nil, err
	}
	idx := distributed.NewTipSetIndexer(queue.NewAsynq(qcfg))

	watchJob := watch.NewWatcher(wapi, idx, cfg.JobConfig.Name,
		watch.WithTasks(cfg.JobConfig.Tasks...),
		watch.WithConfidence(cfg.Confidence),
		watch.WithBufferSize(cfg.BufferSize),
	)

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "watch-notify",
		Params: map[string]string{
			"confidence": strconv.Itoa(cfg.Confidence),
			"buffer":     strconv.Itoa(cfg.BufferSize),
			"queue":      cfg.Queue,
		},
		Tasks:               cfg.JobConfig.Tasks,
		Job:                 watchJob,
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, err
}

func (m *LilyNodeAPI) LilyWalk(_ context.Context, cfg *LilyWalkConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	strg, err := m.StorageCatalog.Connect(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	taskAPI, err := datasource.NewDataSource(m)
	if err != nil {
		return nil, err
	}

	// instantiate an indexer to extract block, message, and actor state data from observed tipsets and persists it to the storage.
	idx, err := integrated.NewManager(taskAPI, strg, cfg.JobConfig.Name, integrated.WithWindow(cfg.JobConfig.Window))
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "walk",
		Params: map[string]string{
			"window":    cfg.JobConfig.Window.String(),
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.JobConfig.Storage,
		},
		Tasks:               cfg.JobConfig.Tasks,
		Job:                 walk.NewWalker(idx, m, cfg.JobConfig.Name, cfg.JobConfig.Tasks, cfg.From, cfg.To),
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyWalkNotify(_ context.Context, cfg *LilyWalkNotifyConfig) (*schedule.JobSubmitResult, error) {
	qcfg, err := m.QueueCatalog.AsynqConfig(cfg.Queue)
	if err != nil {
		return nil, err
	}
	idx := distributed.NewTipSetIndexer(queue.NewAsynq(qcfg))

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.WalkConfig.JobConfig.Name,
		Type: "walk-notify",
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.WalkConfig.From),
			"maxHeight": fmt.Sprintf("%d", cfg.WalkConfig.To),
			"queue":     cfg.Queue,
		},
		Tasks:               cfg.WalkConfig.JobConfig.Tasks,
		Job:                 walk.NewWalker(idx, m, cfg.WalkConfig.JobConfig.Name, cfg.WalkConfig.JobConfig.Tasks, cfg.WalkConfig.From, cfg.WalkConfig.To),
		RestartOnFailure:    cfg.WalkConfig.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.WalkConfig.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.WalkConfig.JobConfig.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyGapFind(_ context.Context, cfg *LilyGapFindConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name:  cfg.JobConfig.Name,
		Type:  "find",
		Tasks: cfg.JobConfig.Tasks,
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.JobConfig.Storage,
		},
		Job:                 gap.NewFinder(m, db, cfg.JobConfig.Name, cfg.From, cfg.To, cfg.JobConfig.Tasks),
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyGapFill(_ context.Context, cfg *LilyGapFillConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.JobConfig.Name,
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "fill",
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.From),
			"maxHeight": fmt.Sprintf("%d", cfg.To),
			"storage":   cfg.JobConfig.Storage,
		},
		Tasks:               cfg.JobConfig.Tasks,
		Job:                 gap.NewFiller(m, db, cfg.JobConfig.Name, cfg.From, cfg.To, cfg.JobConfig.Tasks),
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, nil
}

func (m *LilyNodeAPI) LilyGapFillNotify(_ context.Context, cfg *LilyGapFillNotifyConfig) (*schedule.JobSubmitResult, error) {
	// the context's passed to these methods live for the duration of the clients request, so make a new one.
	ctx := context.Background()

	md := storage.Metadata{
		JobName: cfg.GapFillConfig.JobConfig.Name,
	}

	qcfg, err := m.QueueCatalog.AsynqConfig(cfg.Queue)
	if err != nil {
		return nil, err
	}

	// create a database connection for this watch, ensure its pingable, and run migrations if needed/configured to.
	db, err := m.StorageCatalog.ConnectAsDatabase(ctx, cfg.GapFillConfig.JobConfig.Storage, md)
	if err != nil {
		return nil, err
	}
	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.GapFillConfig.JobConfig.Name,
		Type: "fill-notify",
		Params: map[string]string{
			"minHeight": fmt.Sprintf("%d", cfg.GapFillConfig.From),
			"maxHeight": fmt.Sprintf("%d", cfg.GapFillConfig.To),
			"storage":   cfg.GapFillConfig.JobConfig.Storage,
			"queue":     cfg.Queue,
		},
		Tasks:               cfg.GapFillConfig.JobConfig.Tasks,
		Job:                 gap.NewNotifier(m, db, queue.NewAsynq(qcfg), cfg.GapFillConfig.JobConfig.Name, cfg.GapFillConfig.From, cfg.GapFillConfig.To, cfg.GapFillConfig.JobConfig.Tasks),
		RestartOnFailure:    cfg.GapFillConfig.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.GapFillConfig.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.GapFillConfig.JobConfig.RestartDelay,
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

// TODO pass context to fix wait
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
	strg, err := m.StorageCatalog.Connect(ctx, cfg.JobConfig.Storage, storage.Metadata{JobName: cfg.JobConfig.Name})
	if err != nil {
		return nil, err
	}

	// instantiate a new surveyer.
	surv, err := network.NewSurveyer(m, strg, cfg.Interval, cfg.JobConfig.Name, cfg.JobConfig.Tasks)
	if err != nil {
		return nil, err
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name:  cfg.JobConfig.Name,
		Tasks: cfg.JobConfig.Tasks,
		Job:   surv,
		Params: map[string]string{
			"interval": cfg.Interval.String(),
		},
		RestartOnFailure:    cfg.JobConfig.RestartOnFailure,
		RestartOnCompletion: cfg.JobConfig.RestartOnCompletion,
		RestartDelay:        cfg.JobConfig.RestartDelay,
	})

	return res, nil
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
