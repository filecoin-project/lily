package chain

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
	"golang.org/x/sync/singleflight"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/tasks"
)

var _ tasks.DataSource = (*DataSource)(nil)

type DataSource struct {
	node lens.API

	executedBlkMsgCache *lru.Cache
	executedBlkMsgGroup singleflight.Group

	executedTsCache *lru.Cache
	executedTsGroup singleflight.Group
}

func NewDataSource(node lens.API) (*DataSource, error) {
	t := &DataSource{
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

func (t *DataSource) TipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	return t.node.ChainGetTipSet(ctx, tsk)
}

func (t *DataSource) Store() adt.Store {
	return t.node.Store()
}

func (t *DataSource) Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	return t.node.StateGetActor(ctx, addr, tsk)
}

func (t *DataSource) MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error) {
	return t.node.StateMinerPower(ctx, addr, ts.Key())
}

func (t *DataSource) ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error) {
	return t.node.StateReadState(ctx, addr, ts.Key())
}

func (t *DataSource) ActorStateChanges(ctx context.Context, ts, pts *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	return GetActorStateChanges(ctx, t.Store(), ts, pts)
}

func (t *DataSource) CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error) {
	return t.node.CirculatingSupply(ctx, ts.Key())
}

func (t *DataSource) MessageExecutions(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error) {
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

func (t *DataSource) ExecutedAndBlockMessages(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
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

func GetActorStateChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	if executed.Height() == 0 {
		return GetGenesisActors(ctx, store, executed)
	}

	oldTree, err := state.LoadStateTree(store, executed.ParentState())
	if err != nil {
		return nil, err
	}
	oldTreeRoot, err := oldTree.Flush(ctx)
	if err != nil {
		return nil, err
	}

	newTree, err := state.LoadStateTree(store, current.ParentState())
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
		//log.Warnw("failed to diff state tree efficiently, falling back to slow method", "error", err)
	}
	actors, err := state.Diff(ctx, oldTree, newTree)
	if err != nil {
		return nil, err
	}

	out := map[address.Address]tasks.ActorStateChange{}
	for addrStr, act := range actors {
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return nil, err
		}
		out[addr] = tasks.ActorStateChange{
			Actor:      act,
			ChangeType: tasks.ChangeTypeUnknown,
		}
	}
	return out, nil
}

func GetGenesisActors(ctx context.Context, store adt.Store, genesis *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	out := map[address.Address]tasks.ActorStateChange{}
	tree, err := state.LoadStateTree(store, genesis.ParentState())
	if err != nil {
		return nil, err
	}
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		out[addr] = tasks.ActorStateChange{
			Actor:      *act,
			ChangeType: tasks.ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func fastDiff(ctx context.Context, store adt.Store, oldR, newR cid.Cid) (tasks.ActorStateChangeDiff, error) {
	// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
	changes, err := hamt.Diff(ctx, store, store, oldR, newR, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}))
	if err == nil {
		buf := bytes.NewReader(nil)
		out := map[address.Address]tasks.ActorStateChange{}
		for _, change := range changes {
			addr, err := address.NewFromBytes([]byte(change.Key))
			if err != nil {
				return nil, xerrors.Errorf("address in state tree was not valid: %w", err)
			}
			var ch tasks.ActorStateChange
			switch change.Type {
			case hamt.Add:
				ch.ChangeType = tasks.ChangeTypeAdd
				buf.Reset(change.After.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Remove:
				ch.ChangeType = tasks.ChangeTypeRemove
				buf.Reset(change.Before.Raw)
				err = ch.Actor.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
			case hamt.Modify:
				ch.ChangeType = tasks.ChangeTypeModify
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
	return nil, err
}
