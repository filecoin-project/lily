package test

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/lens"
)

type MockActorStateAPI struct {
	mock.Mock
}

func (m *MockActorStateAPI) TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error) {
	args := m.Called(ctx, ts, pts)
	tsmsgs := args.Get(0)
	err := args.Error(1)
	return tsmsgs.([]*lens.BlockMessageReceipts), err
}

func (m *MockActorStateAPI) MinerLoad(store adt.Store, act *types.Actor) (miner.State, error) {
	args := m.Called(store, act)
	state := args.Get(0)
	err := args.Error(1)
	if state == nil {
		return nil, err
	}
	return state.(miner.State), err
}

func (m *MockActorStateAPI) Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	args := m.Called(ctx, addr, tsk)
	act := args.Get(0)
	err := args.Error(1)
	if act == nil {
		return nil, err
	}
	return act.(*types.Actor), err
}

func (m *MockActorStateAPI) MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error) {
	args := m.Called(ctx, addr, ts)
	power := args.Get(0)
	err := args.Error(1)
	return power.(*api.MinerPower), err
}

func (m *MockActorStateAPI) ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error) {
	args := m.Called(ctx, addr, ts)
	actstate := args.Get(0)
	err := args.Error(1)
	return actstate.(*api.ActorState), err
}

func (m *MockActorStateAPI) Store() adt.Store {
	m.Called()
	return nil
}

func (m *MockActorStateAPI) DiffPreCommits(ctx context.Context, _ address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChanges, error) {
	args := m.Called(ctx, ts, pts, pre, cur)
	tsmsgs := args.Get(0)
	err := args.Error(1)
	return tsmsgs.(*miner.PreCommitChanges), err
}

func (m *MockActorStateAPI) DiffPreCommitsV8(ctx context.Context, _ address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChangesV8, error) {
	args := m.Called(ctx, ts, pts, pre, cur)
	tsmsgs := args.Get(0)
	err := args.Error(1)
	return tsmsgs.(*miner.PreCommitChangesV8), err
}

func (m *MockActorStateAPI) DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error) {
	args := m.Called(ctx, addr, ts, pts, pre, cur)
	changes := args.Get(0)
	err := args.Error(1)
	return changes.(*miner.SectorChanges), err
}
