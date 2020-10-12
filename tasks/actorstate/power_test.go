package actorstate

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/types"
	powermodel "github.com/filecoin-project/sentinel-visor/model/actors/power"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa0smoothing "github.com/filecoin-project/specs-actors/actors/util/smoothing"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa2smoothing "github.com/filecoin-project/specs-actors/v2/actors/util/smoothing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPowerExtractV0(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI()

	state, err := mapi.newEmptyPowerStateV0()
	require.NoError(t, err)

	state.TotalRawBytePower = abi.NewStoragePower(1000)
	state.TotalBytesCommitted = abi.NewStoragePower(2000)
	state.TotalQualityAdjPower = abi.NewStoragePower(3000)
	state.TotalQABytesCommitted = abi.NewStoragePower(4000)
	state.TotalPledgeCollateral = abi.NewTokenAmount(5000)
	state.ThisEpochRawBytePower = abi.NewStoragePower(6000)
	state.ThisEpochQualityAdjPower = abi.NewStoragePower(7000)
	state.ThisEpochPledgeCollateral = abi.NewTokenAmount(8000)
	state.ThisEpochQAPowerSmoothed = sa0smoothing.NewEstimate(big.NewInt(800_000_000_000), big.NewInt(6_000_000_000))
	state.MinerCount = 10
	state.MinerAboveMinPowerCount = 20

	stateCid, err := mapi.Store().Put(ctx, state)
	require.NoError(t, err)

	minerAddr, err := address.NewFromString("t00")
	require.NoError(t, err)
	stateTs, err := mockTipset(minerAddr, 1)
	require.NoError(t, err)

	info := ActorInfo{
		Actor:   types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: stateCid},
		Address: power.Address,
		TipSet:  stateTs.Key(),
	}

	mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: stateCid})

	ex := StoragePowerExtractor{}
	res, err := ex.Extract(ctx, info, mapi)
	require.NoError(t, err)

	cp, ok := res.(*powermodel.ChainPower)
	require.True(t, ok)
	require.NotNil(t, cp)

	assert.EqualValues(t, info.ParentStateRoot.String(), cp.StateRoot, "StateRoot")
	assert.EqualValues(t, state.TotalRawBytePower.String(), cp.TotalRawBytesPower, "TotalRawBytesPower")
	assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.TotalQABytesPower, "TotalQABytesPower")
	assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.TotalRawBytesCommitted, "TotalRawBytesCommitted")
	assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.TotalQABytesCommitted, "TotalQABytesCommitted")
	assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.TotalPledgeCollateral, "TotalPledgeCollateral")
	assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
	assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
	assert.EqualValues(t, state.MinerCount, cp.MinerCount, "MinerCount")
	assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ParticipatingMinerCount, "ParticipatingMinerCount")
}

func TestPowerExtractV2(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI()

	state, err := mapi.newEmptyPowerStateV2()
	require.NoError(t, err)

	state.TotalRawBytePower = abi.NewStoragePower(1000)
	state.TotalBytesCommitted = abi.NewStoragePower(0)
	state.TotalQualityAdjPower = abi.NewStoragePower(0)
	state.TotalQABytesCommitted = abi.NewStoragePower(0)
	state.TotalPledgeCollateral = abi.NewTokenAmount(0)
	state.ThisEpochRawBytePower = abi.NewStoragePower(0)
	state.ThisEpochQualityAdjPower = abi.NewStoragePower(0)
	state.ThisEpochPledgeCollateral = abi.NewTokenAmount(0)
	state.ThisEpochQAPowerSmoothed = sa2smoothing.NewEstimate(big.NewInt(800_000_000_000), big.NewInt(6_000_000_000))
	state.MinerCount = 0
	state.MinerAboveMinPowerCount = 0

	stateCid, err := mapi.Store().Put(ctx, state)
	require.NoError(t, err)

	minerAddr, err := address.NewFromString("t00")
	require.NoError(t, err)
	stateTs, err := mockTipset(minerAddr, 1)
	require.NoError(t, err)

	info := ActorInfo{
		Actor:   types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: stateCid},
		Address: power.Address,
		TipSet:  stateTs.Key(),
	}

	mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: stateCid})

	ex := StoragePowerExtractor{}
	res, err := ex.Extract(ctx, info, mapi)
	require.NoError(t, err)

	cp, ok := res.(*powermodel.ChainPower)
	require.True(t, ok)
	require.NotNil(t, cp)

	assert.EqualValues(t, info.ParentStateRoot.String(), cp.StateRoot, "StateRoot")
	assert.EqualValues(t, state.TotalRawBytePower.String(), cp.TotalRawBytesPower, "TotalRawBytesPower")
	assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.TotalQABytesPower, "TotalQABytesPower")
	assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.TotalRawBytesCommitted, "TotalRawBytesCommitted")
	assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.TotalQABytesCommitted, "TotalQABytesCommitted")
	assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.TotalPledgeCollateral, "TotalPledgeCollateral")
	assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
	assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
	assert.EqualValues(t, state.MinerCount, cp.MinerCount, "MinerCount")
	assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ParticipatingMinerCount, "ParticipatingMinerCount")
}
