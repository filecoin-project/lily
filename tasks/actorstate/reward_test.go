package actorstate_test

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/types"
	rewardmodel "github.com/filecoin-project/sentinel-visor/model/actors/reward"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa0smoothing "github.com/filecoin-project/specs-actors/actors/util/smoothing"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa2smoothing "github.com/filecoin-project/specs-actors/v2/actors/util/smoothing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewardExtractV0(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.newEmptyRewardStateV0(abi.NewStoragePower(500))

	state.CumsumBaseline = big.NewInt(1000)
	state.CumsumRealized = big.NewInt(2000)
	state.EffectiveNetworkTime = abi.ChainEpoch(3000)
	state.EffectiveBaselinePower = abi.NewStoragePower(4000)
	state.ThisEpochReward = abi.NewTokenAmount(5000)
	state.ThisEpochRewardSmoothed = sa0smoothing.NewEstimate(big.NewInt(6000), big.NewInt(60))
	state.ThisEpochBaselinePower = abi.NewStoragePower(7000)
	state.TotalMined = abi.NewStoragePower(8000)

	stateCid, err := mapi.Store().Put(ctx, state)
	require.NoError(t, err)

	minerAddr, err := address.NewFromString("t00")
	require.NoError(t, err)
	stateTs, err := mockTipset(minerAddr, 1)
	require.NoError(t, err)

	info := actorstate.ActorInfo{
		Actor:   types.Actor{Code: sa0builtin.RewardActorCodeID, Head: stateCid},
		Address: power.Address,
		TipSet:  stateTs.Key(),
	}

	mapi.setActor(stateTs.Key(), reward.Address, &types.Actor{Code: sa0builtin.RewardActorCodeID, Head: stateCid})

	ex := actorstate.RewardExtractor{}
	res, err := ex.Extract(ctx, info, mapi)
	require.NoError(t, err)

	cr, ok := res.(*rewardmodel.ChainReward)
	require.True(t, ok)
	require.NotNil(t, cr)

	assert.EqualValues(t, info.ParentStateRoot.String(), cr.StateRoot, "StateRoot")
	assert.EqualValues(t, state.CumsumBaseline.String(), cr.CumSumBaseline, "CumSumBaseline")
	assert.EqualValues(t, state.CumsumRealized.String(), cr.CumSumRealized, "CumSumRealized")
	assert.EqualValues(t, state.EffectiveBaselinePower.String(), cr.EffectiveBaselinePower, "EffectiveBaselinePower")
	assert.EqualValues(t, state.ThisEpochBaselinePower.String(), cr.NewBaselinePower, "NewBaselinePower")
	assert.EqualValues(t, state.ThisEpochRewardSmoothed.PositionEstimate.String(), cr.NewRewardSmoothedPositionEstimate, "NewRewardSmoothedPositionEstimate")
	assert.EqualValues(t, state.ThisEpochRewardSmoothed.VelocityEstimate.String(), cr.NewRewardSmoothedVelocityEstimate, "NewRewardSmoothedVelocityEstimate")
	assert.EqualValues(t, state.TotalMined.String(), cr.TotalMinedReward, "TotalMinedReward")
	assert.EqualValues(t, state.ThisEpochReward.String(), cr.NewReward, "NewReward")
	assert.EqualValues(t, state.EffectiveNetworkTime, cr.EffectiveNetworkTime, "EffectiveNetworkTime")
}

func TestRewardExtractV2(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.newEmptyRewardStateV2(abi.NewStoragePower(500))

	state.CumsumBaseline = big.NewInt(1000)
	state.CumsumRealized = big.NewInt(2000)
	state.EffectiveNetworkTime = abi.ChainEpoch(3000)
	state.EffectiveBaselinePower = abi.NewStoragePower(4000)
	state.ThisEpochReward = abi.NewTokenAmount(5000)
	state.ThisEpochRewardSmoothed = sa2smoothing.NewEstimate(big.NewInt(6000), big.NewInt(60))
	state.ThisEpochBaselinePower = abi.NewStoragePower(7000)
	state.TotalStoragePowerReward = abi.NewStoragePower(8000)

	stateCid, err := mapi.Store().Put(ctx, state)
	require.NoError(t, err)

	minerAddr, err := address.NewFromString("t00")
	require.NoError(t, err)
	stateTs, err := mockTipset(minerAddr, 1)
	require.NoError(t, err)

	info := actorstate.ActorInfo{
		Actor:   types.Actor{Code: sa2builtin.RewardActorCodeID, Head: stateCid},
		Address: power.Address,
		TipSet:  stateTs.Key(),
	}

	mapi.setActor(stateTs.Key(), reward.Address, &types.Actor{Code: sa2builtin.RewardActorCodeID, Head: stateCid})

	ex := actorstate.RewardExtractor{}
	res, err := ex.Extract(ctx, info, mapi)
	require.NoError(t, err)

	cr, ok := res.(*rewardmodel.ChainReward)
	require.True(t, ok)
	require.NotNil(t, cr)

	assert.EqualValues(t, info.ParentStateRoot.String(), cr.StateRoot, "StateRoot")
	assert.EqualValues(t, state.CumsumBaseline.String(), cr.CumSumBaseline, "CumSumBaseline")
	assert.EqualValues(t, state.CumsumRealized.String(), cr.CumSumRealized, "CumSumRealized")
	assert.EqualValues(t, state.EffectiveBaselinePower.String(), cr.EffectiveBaselinePower, "EffectiveBaselinePower")
	assert.EqualValues(t, state.ThisEpochBaselinePower.String(), cr.NewBaselinePower, "NewBaselinePower")
	assert.EqualValues(t, state.ThisEpochRewardSmoothed.PositionEstimate.String(), cr.NewRewardSmoothedPositionEstimate, "NewRewardSmoothedPositionEstimate")
	assert.EqualValues(t, state.ThisEpochRewardSmoothed.VelocityEstimate.String(), cr.NewRewardSmoothedVelocityEstimate, "NewRewardSmoothedVelocityEstimate")
	assert.EqualValues(t, state.TotalStoragePowerReward.String(), cr.TotalMinedReward, "TotalMinedReward")
	assert.EqualValues(t, state.ThisEpochReward.String(), cr.NewReward, "NewReward")
	assert.EqualValues(t, state.EffectiveNetworkTime, cr.EffectiveNetworkTime, "EffectiveNetworkTime")
}
