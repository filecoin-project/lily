package lily

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	network2 "github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/impl/net"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"go.uber.org/fx"

	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/gap"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/chain/indexer/integrated"
	"github.com/filecoin-project/lily/chain/indexer/integrated/tipset"
	"github.com/filecoin-project/lily/chain/walk"
	"github.com/filecoin-project/lily/chain/watch"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/lily/modules"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/network"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/lily/storage"
)

var _ lens.API = (*LilyNodeAPI)(nil)
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

func (m *LilyNodeAPI) Host() host.Host {
	return m.RawHost
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

	worker, err := m.QueueCatalog.Worker(cfg.Queue)
	if err != nil {
		return nil, err
	}

	taskAPI, err := datasource.NewDataSource(m)
	if err != nil {
		return nil, err
	}

	im, err := integrated.NewManager(strg, tipset.NewBuilder(taskAPI, cfg.JobConfig.Name))
	if err != nil {
		return nil, err
	}

	handlers := []queue.TaskHandler{tasks.NewIndexHandler(im)}
	// check if queue config contains configuration for gap fill tasks and if it expects the tasks to be processed. This
	// is specified by giving the Fill queue a priority greater than 1.
	priority, ok := worker.ServerConfig.Queues[indexer.Fill.String()]
	if ok {
		if priority > 0 {
			// if gap fill tasks have a priority storage must be a database.
			db, ok := strg.(*storage.Database)
			if !ok {
				return nil, fmt.Errorf("storage type (%T) is unsupported when %s queue is enable", strg, indexer.Fill.String())
			}
			//  add gap fill handler to set of worker handlers.
			handlers = append(handlers, tasks.NewGapFillHandler(im, db))
		}
	}

	res := m.Scheduler.Submit(&schedule.JobConfig{
		Name: cfg.JobConfig.Name,
		Type: "tipset-worker",
		Params: map[string]string{
			"queue":   cfg.Queue,
			"storage": cfg.JobConfig.Storage,
		},
		Job:                 queue.NewAsynqWorker(cfg.JobConfig.Name, worker, handlers...),
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
	im, err := integrated.NewManager(strg, tipset.NewBuilder(taskAPI, cfg.JobConfig.Name), integrated.WithWindow(cfg.JobConfig.Window))
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

	notifier, err := m.QueueCatalog.Notifier(cfg.Queue)
	if err != nil {
		return nil, err
	}

	ts, err := m.ChainGetTipSet(ctx, cfg.IndexConfig.TipSet)
	if err != nil {
		return nil, err
	}

	idx := distributed.NewTipSetIndexer(queue.NewAsynq(notifier))

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
	idx, err := integrated.NewManager(strg, tipset.NewBuilder(taskAPI, cfg.JobConfig.Name), integrated.WithWindow(cfg.JobConfig.Window))
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

	notifier, err := m.QueueCatalog.Notifier(cfg.Queue)
	if err != nil {
		return nil, err
	}
	idx := distributed.NewTipSetIndexer(queue.NewAsynq(notifier))

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
	idx, err := integrated.NewManager(strg, tipset.NewBuilder(taskAPI, cfg.JobConfig.Name), integrated.WithWindow(cfg.JobConfig.Window))
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
	notifier, err := m.QueueCatalog.Notifier(cfg.Queue)
	if err != nil {
		return nil, err
	}
	idx := distributed.NewTipSetIndexer(queue.NewAsynq(notifier))

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

	notifier, err := m.QueueCatalog.Notifier(cfg.Queue)
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
		Job:                 gap.NewNotifier(m, db, queue.NewAsynq(notifier), cfg.GapFillConfig.JobConfig.Name, cfg.GapFillConfig.From, cfg.GapFillConfig.To, cfg.GapFillConfig.JobConfig.Tasks),
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

func (m *LilyNodeAPI) LilyJobWait(ctx context.Context, ID schedule.JobID) (*schedule.JobListResult, error) {
	res, err := m.Scheduler.WaitJob(ctx, ID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *LilyNodeAPI) LilyJobList(_ context.Context) ([]schedule.JobListResult, error) {
	return m.Scheduler.Jobs(), nil
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
		if err == modules.ErrExecutionTraceNotFound {
			// if lily hasn't watched this tipset be applied then we need to compute its execution trace.
			// this will likely be the case for most walk tasks.
			_, err := m.StateManager.ExecutionTraceWithMonitor(ctx, current, msgMonitor)
			if err != nil {
				return nil, fmt.Errorf("failed to compute execution trace for tipset %s: %w", current.Key().String(), err)
			}
			// the above call will populate the msgMonitor with an execution trace for this tipset, get it.
			executions, err = msgMonitor.ExecutionFor(current)
			if err != nil {
				return nil, fmt.Errorf("failed to find execution trace for tipset %s: %w", current.Key().String(), err)
			}
		} else {
			return nil, fmt.Errorf("failed to extract message execution for tipset %s: %w", next, err)
		}
	}

	getActorCode, err := util.MakeGetActorCodeFunc(ctx, m.ChainAPI.Chain.ActorStore(ctx), next, current)
	if err != nil {
		return nil, fmt.Errorf("failed to make actor code query function: %w", err)
	}

	out := make([]*lens.MessageExecution, len(executions))
	for idx, execution := range executions {
		toCode, found := getActorCode(ctx, execution.Msg.To)
		// if the message failed to execute due to lack of gas then the TO actor may never have been created.
		if !found {
			log.Warnw("failed to find TO actor", "height", next.Height().String(), "message", execution.Msg.Cid().String(), "actor", execution.Msg.To.String())
		}
		// if the message sender cannot be found this is an unexpected error
		fromCode, found := getActorCode(ctx, execution.Msg.From)
		if !found {
			return nil, fmt.Errorf("failed to find from actor %s height %d message %s", execution.Msg.From, execution.TipSet.Height(), execution.Msg.Cid())
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

// ComputeBaseFee calculates the base-fee of the specified tipset.
func (m *LilyNodeAPI) ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error) {
	return m.ChainAPI.Chain.ComputeBaseFee(ctx, ts)
}

// MessagesForTipSetBlocks returns messages stored in the blocks of the specified tipset, messages may be duplicated
// across the returned set of BlockMessages.
func (m *LilyNodeAPI) MessagesForTipSetBlocks(ctx context.Context, ts *types.TipSet) ([]*lens.BlockMessages, error) {
	var out []*lens.BlockMessages
	for _, blk := range ts.Blocks() {
		blkMsgs, err := m.ChainAPI.ChainModuleAPI.ChainGetBlockMessages(ctx, blk.Cid())
		if err != nil {
			return nil, err
		}
		out = append(out, &lens.BlockMessages{
			Block:        blk,
			BlsMessages:  blkMsgs.BlsMessages,
			SecpMessages: blkMsgs.SecpkMessages,
		})
	}
	return out, nil
}

// TipSetMessageReceipts returns the blocks and messages in `pts` and their corresponding receipts from `ts` matching block order in tipset (`pts`).
func (m *LilyNodeAPI) TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error) {
	// sanity check args
	if ts.Key().IsEmpty() {
		return nil, fmt.Errorf("tipset cannot be empty")
	}
	if pts.Key().IsEmpty() {
		return nil, fmt.Errorf("parent tipset cannot be empty")
	}
	if !types.CidArrsEqual(ts.Parents().Cids(), pts.Cids()) {
		return nil, fmt.Errorf("mismatching tipset (%s) and parent tipset (%s)", ts.Key().String(), pts.Key().String())
	}
	// returned BlockMessages match block order in tipset
	blkMsgs, err := m.ChainAPI.Chain.BlockMsgsForTipset(ctx, pts)
	if err != nil {
		return nil, err
	}
	if len(blkMsgs) != len(pts.Blocks()) {
		// logic error somewhere
		return nil, fmt.Errorf("mismatching number of blocks returned from block messages, got %d wanted %d", len(blkMsgs), len(pts.Blocks()))
	}

	// retrieve receipts using a block from the child (ts) tipset
	rs, err := adt.AsArray(m.Store(), ts.Blocks()[0].ParentMessageReceipts)
	if err != nil {
		// if we fail to find the receipts then we need to compute them, which we can safely do since the above `BlockMsgsForTipset` call
		// returning successfully indicates we have the message available to compute receipts from.
		if ipld.IsNotFound(err) {
			log.Debugw("computing tipset to get receipts", "ts", pts.Key().String(), "height", pts.Height())
			if stateRoot, receiptRoot, err := m.StateManager.TipSetState(ctx, pts); err != nil {
				log.Errorw("failed to compute tipset state", "tipset", pts.Key().String(), "height", pts.Height())
				return nil, err
			} else if !stateRoot.Equals(ts.ParentState()) { // sanity check
				return nil, fmt.Errorf("computed stateroot (%s) does not match tipset stateroot (%s)", stateRoot.String(), ts.ParentState().String())
			} else if !receiptRoot.Equals(ts.Blocks()[0].ParentMessageReceipts) { // sanity check
				return nil, fmt.Errorf("computed receipts (%s) does not match tipset block parent message receipts (%s)", receiptRoot.String(), ts.Blocks()[0].ParentMessageReceipts.String())
			}
			// loading after computing state should succeed as tipset computation produces message receipts
			rs, err = adt.AsArray(m.Store(), ts.Blocks()[0].ParentMessageReceipts)
			if err != nil {
				return nil, fmt.Errorf("load message receipts after tipset execution (something if very wrong contact a developer): %w", err)
			}
		} else {
			return nil, fmt.Errorf("loading message receipts %w", err)
		}
	}
	// so we only load the receipt array once
	getReceipt := func(idx int) (*types.MessageReceipt, error) {
		var r types.MessageReceipt
		if found, err := rs.Get(uint64(idx), &r); err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("failed to find receipt %d", idx)
		}
		return &r, nil
	}

	out := make([]*lens.BlockMessageReceipts, len(pts.Blocks()))
	executionIndex := 0
	// walk each block in tipset, `pts.Blocks()` has same ordering as `blkMsgs`.
	for blkIdx := range pts.Blocks() {
		// bls and secp messages for block
		msgs := blkMsgs[blkIdx]
		// index of messages in `out.Messages`
		msgIdx := 0
		// index or receipts in `out.Receipts`
		receiptIdx := 0
		out[blkIdx] = &lens.BlockMessageReceipts{
			// block containing messages
			Block: pts.Blocks()[blkIdx],
			// total messages returned equal to sum of bls and secp messages
			Messages: make([]types.ChainMsg, len(msgs.BlsMessages)+len(msgs.SecpkMessages)),
			// total receipts returned equal to sum of bls and secp messages
			Receipts: make([]*types.MessageReceipt, len(msgs.BlsMessages)+len(msgs.SecpkMessages)),
			// index of message indicating execution order.
			MessageExecutionIndex: make(map[types.ChainMsg]int),
		}
		// walk bls messages and extract their receipts
		for blsIdx := range msgs.BlsMessages {
			receipt, err := getReceipt(executionIndex)
			if err != nil {
				return nil, err
			}
			out[blkIdx].Messages[msgIdx] = msgs.BlsMessages[blsIdx]
			out[blkIdx].Receipts[receiptIdx] = receipt
			out[blkIdx].MessageExecutionIndex[msgs.BlsMessages[blsIdx]] = executionIndex
			msgIdx++
			receiptIdx++
			executionIndex++
		}
		// walk secp messages and extract their receipts
		for secpIdx := range msgs.SecpkMessages {
			receipt, err := getReceipt(executionIndex)
			if err != nil {
				return nil, err
			}
			out[blkIdx].Messages[msgIdx] = msgs.SecpkMessages[secpIdx]
			out[blkIdx].Receipts[receiptIdx] = receipt
			out[blkIdx].MessageExecutionIndex[msgs.SecpkMessages[secpIdx]] = executionIndex
			msgIdx++
			receiptIdx++
			executionIndex++
		}
	}
	return out, nil
}

type vmWrapper struct {
	vm vm.Interface
	st *state.StateTree
}

func (v *vmWrapper) ShouldBurn(ctx context.Context, msg *types.Message, errcode exitcode.ExitCode) (bool, error) {
	if lvmi, ok := v.vm.(*vm.LegacyVM); ok {
		return lvmi.ShouldBurn(ctx, v.st, msg, errcode)
	}

	// Any "don't burn" rules from Network v13 onwards go here, for now we always return true
	// source: https://github.com/filecoin-project/lotus/blob/v1.15.1/chain/vm/vm.go#L647
	return true, nil
}

func (m *LilyNodeAPI) BurnFundsFn(ctx context.Context, ts *types.TipSet) (lens.ShouldBurnFn, error) {
	// Create a skeleton vm just for calling ShouldBurn
	// NB: VM is only required to process state prior to network v13
	if util.DefaultNetwork.Version(ctx, ts.Height()) <= network2.Version12 {
		vmi, err := vm.NewVM(ctx, &vm.VMOpts{
			StateBase:      ts.ParentState(),
			Epoch:          ts.Height(),
			Bstore:         m.ChainAPI.Chain.StateBlockstore(),
			Actors:         filcns.NewActorRegistry(),
			Syscalls:       m.StateManager.Syscalls,
			CircSupplyCalc: m.StateManager.GetVMCirculatingSupply,
			NetworkVersion: util.DefaultNetwork.Version(ctx, ts.Height()),
			BaseFee:        ts.Blocks()[0].ParentBaseFee,
		})
		if err != nil {
			return nil, fmt.Errorf("creating temporary vm: %w", err)
		}
		parentStateTree, err := state.LoadStateTree(m.ChainAPI.Chain.ActorStore(ctx), ts.ParentState())
		if err != nil {
			return nil, err
		}
		vmw := &vmWrapper{vm: vmi, st: parentStateTree}
		return vmw.ShouldBurn, nil
	}
	// always burn after Network Version 12
	return func(ctx context.Context, msg *types.Message, errcode exitcode.ExitCode) (bool, error) {
		return true, nil
	}, nil
}

func (m *LilyNodeAPI) CirculatingSupply(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	return m.StateAPI.StateVMCirculatingSupplyInternal(ctx, key)
}

func (m *LilyNodeAPI) ChainGetTipSetAfterHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	// TODO (Frrist): I copied this from lotus, I need it now to handle gap filling edge cases.
	ts, err := m.ChainAPI.Chain.GetTipSetFromKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("loading tipset %s: %w", key, err)
	}
	return m.ChainAPI.Chain.GetTipsetByHeight(ctx, epoch, ts, false)
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

func (m *LilyNodeAPI) LogList(_ context.Context) ([]string, error) {
	return logging.GetSubsystems(), nil
}

func (m *LilyNodeAPI) LogSetLevel(_ context.Context, subsystem, level string) error {
	return logging.SetLogLevel(subsystem, level)
}

func (m *LilyNodeAPI) LogSetLevelRegex(_ context.Context, regex, level string) error {
	return logging.SetLogLevelRegex(regex, level)
}

func (m *LilyNodeAPI) Shutdown(_ context.Context) error {
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

	// instantiate a new survey.
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

type StateReport struct {
	Height      int64
	TipSet      *types.TipSet
	HasMessages bool
	HasReceipts bool
	HasState    bool
}

func (m *LilyNodeAPI) StateCompute(ctx context.Context, key types.TipSetKey) (interface{}, error) {
	ts, err := m.ChainAPI.ChainGetTipSet(ctx, key)
	if err != nil {
		return nil, err
	}
	_, _, err = m.StateManager.TipSetState(ctx, ts)
	return nil, err
}

func (m *LilyNodeAPI) FindOldestState(ctx context.Context, limit int64) ([]*StateReport, error) {
	head, err := m.ChainHead(ctx)
	if err != nil {
		return nil, err
	}

	var out []*StateReport
	var oldestEpochLimit = head.Height() - abi.ChainEpoch(limit)

	for i := int64(0); true; i++ {
		maybeBaseTS, err := m.ChainGetTipSetByHeight(ctx, head.Height()-abi.ChainEpoch(i), head.Key())
		if err != nil {
			return nil, err
		}

		maybeFullTS := TryLoadFullTipSet(ctx, m, maybeBaseTS)
		out = append(out, &StateReport{
			Height:      int64(maybeBaseTS.Height()),
			TipSet:      maybeBaseTS,
			HasMessages: maybeFullTS.HasMessages,
			HasReceipts: maybeFullTS.HasReceipts,
			HasState:    maybeFullTS.HasState,
		})
		if (head.Height() - abi.ChainEpoch(i)) <= oldestEpochLimit {
			break
		}
	}
	return out, nil
}

type FullBlock struct {
	Header       *types.BlockHeader
	BlsMessages  []*types.Message
	SecpMessages []*types.SignedMessage
}

type FullTipSet struct {
	Blocks []*FullBlock
	TipSet *types.TipSet

	HasMessages bool
	HasState    bool
	HasReceipts bool
}

func TryLoadFullTipSet(ctx context.Context, m *LilyNodeAPI, ts *types.TipSet) *FullTipSet {
	var (
		out         []*FullBlock
		err         error
		hasState    = true
		hasMessages = true
		hasReceipts = true
	)

	for _, b := range ts.Blocks() {
		fb := &FullBlock{Header: b}

		fb.BlsMessages, fb.SecpMessages, err = m.ChainAPI.Chain.MessagesForBlock(ctx, b)
		if err != nil {
			log.Debugw("failed to load messages", "height", b.Height)
			hasMessages = false
		}
		out = append(out, fb)
	}

	_, err = adt.AsArray(m.ChainAPI.Chain.ActorStore(ctx), ts.Blocks()[0].ParentMessageReceipts)
	if err != nil {
		log.Debugw("failed to load receipts", "height", ts.Blocks()[0].Height)
		hasReceipts = false
	}

	_, err = m.StateManager.ParentState(ts)
	if err != nil {
		log.Debugw("failed to load statetree", "height", ts.Blocks()[0].Height)
		hasState = false
	}

	return &FullTipSet{
		Blocks:      out,
		TipSet:      ts,
		HasMessages: hasMessages,
		HasState:    hasState,
		HasReceipts: hasReceipts,
	}
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

func (l *LogQueryHook) AfterQuery(_ context.Context, _ *pg.QueryEvent) error {
	return nil
}
