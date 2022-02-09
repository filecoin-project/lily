package task

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/sync/singleflight"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
)

var log = logging.Logger("lily/lens")

type TaskAPI interface {
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)

	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	StateChangedActors(ctx context.Context, store adt.Store, ts, pts *types.TipSet) (ActorStateChangeDiff, error)
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

func (t *TaskAPIImpl) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	return t.node.ChainGetTipSet(ctx, key)
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

func (t *TaskAPIImpl) StateChangedActors(ctx context.Context, store adt.Store, ts, pts *types.TipSet) (ActorStateChangeDiff, error) {
	return GetActorStateChanges(ctx, store, ts, pts)
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

// TODO(frrist): instrument this method with logs for tracking its duration and cache hits v misses
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

// ChangeType denotes type of state change
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

type ActorStateChange struct {
	Actor      types.Actor
	ChangeType ChangeType
}

type ActorStateChangeDiff map[string]ActorStateChange

func GetActorStateChanges(ctx context.Context, store adt.Store, current, next *types.TipSet) (ActorStateChangeDiff, error) {
	if current.Height() == 0 {
		return GetGenesisActors(ctx, store, current)
	}

	oldTree, err := state.LoadStateTree(store, current.ParentState())
	if err != nil {
		return nil, err
	}
	oldTreeRoot, err := oldTree.Flush(ctx)
	if err != nil {
		return nil, err
	}

	newTree, err := state.LoadStateTree(store, next.ParentState())
	if err != nil {
		return nil, err
	}
	newTreeRoot, err := newTree.Flush(ctx)
	if err != nil {
		return nil, err
	}

	if newTree.Version() == oldTree.Version() && (newTree.Version() != types.StateTreeVersion0 && newTree.Version() != types.StateTreeVersion1) {
		changes, err := fastDiff(ctx, store, oldTreeRoot, newTreeRoot)
		if err == nil {
			return changes, nil
		}
		log.Warnw("failed to diff state tree efficiently, falling back to slow method", "error", err)
	}
	actors, err := state.Diff(ctx, oldTree, newTree)
	if err != nil {
		return nil, err
	}

	out := map[string]ActorStateChange{}
	for addr, act := range actors {
		out[addr] = ActorStateChange{
			Actor:      act,
			ChangeType: ChangeTypeUnknown,
		}
	}
	return out, nil
}

func GetGenesisActors(ctx context.Context, store adt.Store, genesis *types.TipSet) (ActorStateChangeDiff, error) {
	out := map[string]ActorStateChange{}
	tree, err := state.LoadStateTree(store, genesis.ParentState())
	if err != nil {
		return nil, err
	}
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		out[addr.String()] = ActorStateChange{
			Actor:      *act,
			ChangeType: ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func fastDiff(ctx context.Context, store adt.Store, oldR, newR cid.Cid) (ActorStateChangeDiff, error) {
	// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
	changes, err := hamt.Diff(ctx, store, store, oldR, newR, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}))
	if err == nil {
		buf := bytes.NewReader(nil)
		out := map[string]ActorStateChange{}
		for _, change := range changes {
			addr, err := address.NewFromBytes([]byte(change.Key))
			if err != nil {
				return nil, xerrors.Errorf("address in state tree was not valid: %w", err)
			}
			var ch ActorStateChange
			switch change.Type {
			case hamt.Add:
				ch.ChangeType = ChangeTypeAdd
				buf.Reset(change.After.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Remove:
				ch.ChangeType = ChangeTypeRemove
				buf.Reset(change.Before.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Modify:
				ch.ChangeType = ChangeTypeModify
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
	return nil, err
}
