package actorstate_test

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonmodel "github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

func TestActorExtractor(t *testing.T) {
	ctx := context.Background()
	mapi := NewMockAPI(t)

	expectedAddress := tutils.NewIDAddr(t, 123)
	state := mapi.mustCreateAccountStateV0(expectedAddress)
	expectedHead, err := mapi.Store().Put(ctx, state)
	require.NoError(t, err)
	expectedCode := builtin.AccountActorCodeID
	expectedNonce := uint64(1)
	expectedBal := types.NewInt(1)

	act := types.Actor{
		Code:    expectedCode,
		Head:    expectedHead,
		Nonce:   expectedNonce,
		Balance: expectedBal,
	}

	minerAddr := tutils.NewIDAddr(t, 1234)
	tipset := mapi.fakeTipset(minerAddr, 1)
	mapi.setActor(tipset.Key(), expectedAddress, &act)

	expectedHieght := abi.ChainEpoch(1)
	info := actorstate.ActorInfo{
		Actor:   act,
		Address: expectedAddress,
		TipSet:  tipset,
	}

	ex := actorstate.ActorExtractor{}
	res, err := ex.Extract(ctx, info, mapi)
	assert.NoError(t, err)

	actualState, ok := res.(*commonmodel.ActorTaskResult)
	assert.True(t, ok)
	assert.NotNil(t, actualState)

	assert.EqualValues(t, expectedCode.String(), actualState.State.Code)
	assert.EqualValues(t, expectedHieght, actualState.State.Height)
	assert.EqualValues(t, expectedHead.String(), actualState.State.Head)

	assert.EqualValues(t, expectedHead.String(), actualState.Actor.Head)
	assert.EqualValues(t, expectedHieght, actualState.Actor.Height)
	assert.EqualValues(t, builtin.ActorNameByCode(expectedCode), actualState.Actor.Code)
	assert.EqualValues(t, expectedAddress.String(), actualState.Actor.ID)
	assert.EqualValues(t, expectedNonce, actualState.Actor.Nonce)
	assert.EqualValues(t, expectedBal.String(), actualState.Actor.Balance)
	assert.EqualValues(t, tipset.ParentState().String(), actualState.Actor.StateRoot)
}
