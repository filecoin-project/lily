package extraction

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	minerstatemocks "github.com/filecoin-project/lily/chain/actors/builtin/miner/mocks"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	atesting "github.com/filecoin-project/lily/tasks/test"
	"github.com/filecoin-project/lily/testutil"

	"github.com/filecoin-project/lotus/chain/types"
)

func TestLoadMinerStatesPreviousStatePresent(t *testing.T) {
	ctx := context.Background()
	currentActor := types.Actor{
		Code:    cid.Undef,
		Head:    cid.Undef,
		Nonce:   0,
		Balance: types.BigInt{},
	}
	parentActor := types.Actor{
		Code:    cid.Undef,
		Head:    cid.Undef,
		Nonce:   1,
		Balance: types.BigInt{},
	}
	currentTs := testutil.MustFakeTipSet(t, 10)
	parentTs := testutil.MustFakeTipSet(t, 9)
	addr := testutil.MustMakeAddress(t, 111)
	actorInfo := actorstate.ActorInfo{
		Actor:      currentActor,
		ChangeType: tasks.ChangeTypeUnknown,
		Address:    addr,
		Current:    currentTs,
		Executed:   parentTs,
	}

	mActorStateAPI := new(atesting.MockActorStateAPI)
	mCurrentState := new(minerstatemocks.State)
	mParentState := new(minerstatemocks.State)
	mActorStateAPI.On("Store").Return(mock.Anything)
	mActorStateAPI.On("MinerLoad", mock.Anything, &currentActor).Return(mCurrentState, nil)
	mActorStateAPI.On("Actor", mock.Anything, actorInfo.Address, actorInfo.Executed.Key()).Return(&parentActor, nil)
	mActorStateAPI.On("Store").Return(mock.Anything)
	mActorStateAPI.On("MinerLoad", mock.Anything, &parentActor).Return(mParentState, nil)
	loadedState, err := LoadMinerStates(ctx, actorInfo, mActorStateAPI)
	require.NoError(t, err)
	require.NotNil(t, loadedState)
	require.Equal(t, mParentState, loadedState.parentState)
	require.Equal(t, parentTs, loadedState.parentTipSet)
	require.Equal(t, mCurrentState, loadedState.currentState)
	require.Equal(t, currentTs, loadedState.currentTipset)
	require.Equal(t, currentActor, loadedState.actor)
	require.Equal(t, addr, loadedState.address)
	require.Equal(t, tasks.ChangeTypeUnknown, loadedState.changeType)
	require.Equal(t, mActorStateAPI, loadedState.api)
}

func TestLoadMinerStatesNoPreviousStatePresent(t *testing.T) {
	ctx := context.Background()
	currentActor := types.Actor{
		Code:    cid.Undef,
		Head:    cid.Undef,
		Nonce:   0,
		Balance: types.BigInt{},
	}
	currentTs := testutil.MustFakeTipSet(t, 10)
	parentTs := testutil.MustFakeTipSet(t, 9)
	addr := testutil.MustMakeAddress(t, 111)
	actorInfo := actorstate.ActorInfo{
		Actor:      currentActor,
		ChangeType: tasks.ChangeTypeUnknown,
		Address:    addr,
		Current:    currentTs,
		Executed:   parentTs,
	}

	mActorStateAPI := new(atesting.MockActorStateAPI)
	mCurrentState := new(minerstatemocks.State)
	mActorStateAPI.On("Store").Return(mock.Anything)
	mActorStateAPI.On("MinerLoad", mock.Anything, &currentActor).Return(mCurrentState, nil)
	mActorStateAPI.On("Actor", mock.Anything, actorInfo.Address, actorInfo.Executed.Key()).Return(nil, types.ErrActorNotFound)
	mActorStateAPI.On("Store").Return(mock.Anything)
	loadedState, err := LoadMinerStates(ctx, actorInfo, mActorStateAPI)
	require.NoError(t, err)
	require.NotNil(t, loadedState)
	require.Equal(t, nil, loadedState.parentState)
	require.Equal(t, parentTs, loadedState.parentTipSet)
	require.Equal(t, mCurrentState, loadedState.currentState)
	require.Equal(t, currentTs, loadedState.currentTipset)
	require.Equal(t, currentActor, loadedState.actor)
	require.Equal(t, addr, loadedState.address)
	require.Equal(t, tasks.ChangeTypeUnknown, loadedState.changeType)
	require.Equal(t, mActorStateAPI, loadedState.api)
}
