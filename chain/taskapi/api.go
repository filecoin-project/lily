package taskapi

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sync/singleflight"
)

type TaskAPI interface {
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	Store() adt.Store

	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)

	// memoized methods
	GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
	GetMessageExecutionsForTipSet(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error)
}

var _ TaskAPI = (*TaskAPIImpl)(nil)

type TaskAPIImpl struct {
	node lens.API

	executedBlkMsgCache *lru.Cache
	executedBlkMsgGroup singleflight.Group

	executedTsCache *lru.Cache
	executedTsGroup singleflight.Group
}

func NewTaskAPI(node lens.API) (*TaskAPIImpl, error) {
	t := &TaskAPIImpl{
		node:                node,
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
	return t.node.Store()
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
