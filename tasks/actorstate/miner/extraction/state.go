package extraction

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type State interface {
	CurrentState() miner.State
	CurrentTipSet() *types.TipSet

	ParentState() miner.State
	ParentTipSet() *types.TipSet

	Actor() types.Actor
	Address() address.Address

	ChangeType() tasks.ChangeType

	API() actorstate.ActorStateAPI
}

func LoadMinerStates(ctx context.Context, a actorstate.ActorInfo, api actorstate.ActorStateAPI) (*MinerState, error) {
	currentState, err := api.MinerLoad(api.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current miner state: %w", err)
	}

	parentActor, err := api.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// if the actor exists in the current state and not in the parent state then the
		// actor was created in the current state.
		if err == types.ErrActorNotFound {
			return &MinerState{
				parentState:   nil, // since there is no previous state
				parentTipSet:  a.Executed,
				currentState:  currentState,
				currentTipset: a.Current,
				actor:         a.Actor,
				address:       a.Address,
				changeType:    a.ChangeType,
				api:           api,
			}, nil
		}
		return nil, fmt.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.Executed.Key(), a.Current.Height(), err)
	}

	previousState, err := api.MinerLoad(api.Store(), parentActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous miner actor state: %w", err)
	}

	return &MinerState{
		parentState:   previousState,
		parentTipSet:  a.Executed,
		currentState:  currentState,
		currentTipset: a.Current,
		actor:         a.Actor,
		address:       a.Address,
		changeType:    a.ChangeType,
		api:           api,
	}, nil
}

type MinerState struct {
	parentState  miner.State
	parentTipSet *types.TipSet

	currentState  miner.State
	currentTipset *types.TipSet

	actor   types.Actor
	address address.Address

	changeType tasks.ChangeType

	api actorstate.ActorStateAPI
}

func (m *MinerState) CurrentState() miner.State {
	return m.currentState
}

func (m *MinerState) CurrentTipSet() *types.TipSet {
	return m.currentTipset
}

func (m *MinerState) ParentState() miner.State {
	return m.parentState
}

func (m *MinerState) ParentTipSet() *types.TipSet {
	return m.parentTipSet
}

func (m *MinerState) Actor() types.Actor {
	return m.actor
}

func (m *MinerState) Address() address.Address {
	return m.address
}

func (m *MinerState) ChangeType() tasks.ChangeType {
	return m.changeType
}

func (m *MinerState) API() actorstate.ActorStateAPI {
	return m.api
}
