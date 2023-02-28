package mocks

import (
	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type MockMinerState struct {
	mock.Mock
}

func (m *MockMinerState) CurrentState() miner.State {
	args := m.Called()
	maybeState := args.Get(0)
	if maybeState == nil {
		return nil
	}
	return maybeState.(miner.State)
}

func (m *MockMinerState) CurrentTipSet() *types.TipSet {
	args := m.Called()
	return args.Get(0).(*types.TipSet)
}

func (m *MockMinerState) ParentState() miner.State {
	args := m.Called()
	maybeState := args.Get(0)
	if maybeState == nil {
		return nil
	}
	return maybeState.(miner.State)
}

func (m *MockMinerState) ParentTipSet() *types.TipSet {
	args := m.Called()
	return args.Get(0).(*types.TipSet)
}

func (m *MockMinerState) Actor() types.Actor {
	args := m.Called()
	return args.Get(0).(types.Actor)
}

func (m *MockMinerState) Address() address.Address {
	args := m.Called()
	return args.Get(0).(address.Address)
}

func (m *MockMinerState) ChangeType() tasks.ChangeType {
	args := m.Called()
	return args.Get(0).(tasks.ChangeType)
}

func (m *MockMinerState) API() actorstate.ActorStateAPI {
	args := m.Called()
	return args.Get(0).(actorstate.ActorStateAPI)
}
