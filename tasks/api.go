package tasks

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/adt"
	builtin2 "github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/singleflight"
)

var log = logging.Logger("lily/tasks/api")

type TaskAPI interface {
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	StateChangedActors(ctx context.Context, store adt.Store, ts, pts *types.TipSet) (util.ActorStateChangeDiff, error)
	Store() adt.Store
	WarmStoreCache(ts *types.TipSet)

	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)

	// memoized methods
	GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
	GetMessageExecutionsForTipSet(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error)
}

var _ TaskAPI = (*TaskAPIImpl)(nil)

type TaskAPIImpl struct {
	node  lens.API
	store *TaskStore

	executedBlkMsgCache *lru.Cache
	executedBlkMsgGroup singleflight.Group

	executedTsCache *lru.Cache
	executedTsGroup singleflight.Group
}

func NewTaskAPI(node lens.API, cacheSize int) (*TaskAPIImpl, error) {
	store := NewTaskStore(node.ChainBlockstore(), node.StateBlockstore(), cacheSize)
	t := &TaskAPIImpl{
		node:                node,
		store:               store,
		executedBlkMsgGroup: singleflight.Group{},
		executedTsGroup:     singleflight.Group{},
	}
	blkMsgCache, err := lru.New(4)
	if err != nil {
		return nil, err
	}
	t.executedBlkMsgCache = blkMsgCache

	tsCache, err := lru.New(4)
	if err != nil {
		return nil, err
	}
	t.executedTsCache = tsCache

	return t, nil
}

func (t *TaskAPIImpl) Store() adt.Store {
	return t.store
}

func (t *TaskAPIImpl) WarmStoreCache(ts *types.TipSet) {
	tree, err := state.LoadStateTree(t.store, ts.ParentState())
	if err != nil {
		log.Warnw("failed to warm TaskAPI store", "error", err)
		return
	}
	if err := tree.ForEach(func(a address.Address, actor *types.Actor) error {
		go func() {
			switch builtin2.ActorFamily(builtin2.ActorNameByCode(actor.Code)) {
			case "storageminer":
				m, err := miner.Load(t.store, actor)
				if err != nil {
					log.Warnw("failed to load miner while warming TaskAPI store", "error", err)
				}
				m.LoadSectors(nil)
			}
		}()
		return nil
	}); err != nil {
		log.Warnw("failed to walk state tree while warming TaskAPI store", "error", err)
	}
}

func (t *TaskAPIImpl) StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return t.node.StateGetActor(ctx, addr, tsk)
}

func (t *TaskAPIImpl) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	return t.node.StateMinerPower(ctx, addr, tsk)
}

func (t *TaskAPIImpl) StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	return t.node.StateReadState(ctx, addr, tsk)
}

func (t *TaskAPIImpl) StateChangedActors(ctx context.Context, store adt.Store, ts, pts *types.TipSet) (util.ActorStateChangeDiff, error) {
	return util.GetActorStateChanges(ctx, store, ts, pts)
}

func (t *TaskAPIImpl) StateVMCirculatingSupplyInternal(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	return t.node.StateVMCirculatingSupplyInternal(ctx, key)
}

func (t *TaskAPIImpl) GetMessageExecutionsForTipSet(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error) {
	key := ts.Key().String() + pts.Key().String()
	value, found := t.executedTsCache.Get(key)
	if found {
		return value.([]*lens.MessageExecution), nil
	}

	value, err, _ := t.executedTsGroup.Do(key, func() (interface{}, error) {
		data, innerErr := t.node.GetMessageExecutionsForTipSet(ctx, ts, pts)
		if innerErr == nil {
			t.executedTsCache.Add(key, data)
		}

		return data, innerErr
	})
	if err != nil {
		return nil, err
	}
	return value.([]*lens.MessageExecution), nil
}

func (t *TaskAPIImpl) GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
	key := ts.Key().String() + pts.Key().String()
	value, found := t.executedBlkMsgCache.Get(key)
	if found {
		return value.(*lens.TipSetMessages), nil
	}

	value, err, _ := t.executedBlkMsgGroup.Do(key, func() (interface{}, error) {
		data, innerErr := t.node.GetExecutedAndBlockMessagesForTipset(ctx, ts, pts)
		if innerErr == nil {
			t.executedBlkMsgCache.Add(key, data)
		}

		return data, innerErr
	})
	if err != nil {
		return nil, err
	}
	return value.(*lens.TipSetMessages), nil
}
