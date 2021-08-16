package actorstate_test

import (
	"context"
	"testing"

	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/power"

	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	power0 "github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	sa0smoothing "github.com/filecoin-project/specs-actors/actors/util/smoothing"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	sa2smoothing "github.com/filecoin-project/specs-actors/v2/actors/util/smoothing"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	sa3smoothing "github.com/filecoin-project/specs-actors/v3/actors/util/smoothing"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	sa4smoothing "github.com/filecoin-project/specs-actors/v4/actors/util/smoothing"

	"github.com/filecoin-project/sentinel-visor/lens"
	powermodel "github.com/filecoin-project/sentinel-visor/model/actors/power"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
)

func TestPowerExtractV0(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.mustCreateEmptyPowerStateV0()
	minerAddr := tutils.NewIDAddr(t, 1234)

	t.Run("power state model", func(t *testing.T) {
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

		stateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: stateCid})

		info := actorstate.ActorInfo{
			Actor:           types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: stateCid},
			Address:         power.Address,
			TipSet:          stateTs,
			ParentStateRoot: stateTs.ParentState(),
			Epoch:           1,
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.EqualValues(t, info.ParentStateRoot.String(), cp.ChainPowerModel.StateRoot, "StateRoot")
		assert.EqualValues(t, state.TotalRawBytePower.String(), cp.ChainPowerModel.TotalRawBytesPower, "TotalRawBytesPower")
		assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.ChainPowerModel.TotalQABytesPower, "TotalQABytesPower")
		assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.ChainPowerModel.TotalRawBytesCommitted, "TotalRawBytesCommitted")
		assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.ChainPowerModel.TotalQABytesCommitted, "TotalQABytesCommitted")
		assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.ChainPowerModel.TotalPledgeCollateral, "TotalPledgeCollateral")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.ChainPowerModel.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.ChainPowerModel.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
		assert.EqualValues(t, state.MinerCount, cp.ChainPowerModel.MinerCount, "MinerCount")
		assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ChainPowerModel.ParticipatingMinerCount, "ParticipatingMinerCount")
	})

	t.Run("power claim model", func(t *testing.T) {
		claimMap, err := adt.AsMap(mapi.store, state.Claims)
		require.NoError(t, err)
		newClaim := power0.Claim{
			RawBytePower:    abi.NewStoragePower(10),
			QualityAdjPower: abi.NewStoragePower(5),
		}

		oldStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		oldStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(oldStateTs.Key(), power.Address, &types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: oldStateCid})

		err = claimMap.Put(abi.AddrKey(minerAddr), &newClaim)
		require.NoError(t, err)

		state.Claims, err = claimMap.Root()
		require.NoError(t, err)

		newStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		newStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(newStateTs.Key(), power.Address, &types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: newStateCid})

		info := actorstate.ActorInfo{
			Epoch:           1,
			Actor:           types.Actor{Code: sa0builtin.StoragePowerActorCodeID, Head: newStateCid},
			Address:         power.Address,
			ParentTipSet:    oldStateTs,
			TipSet:          newStateTs,
			ParentStateRoot: newStateTs.ParentState(),
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.Len(t, cp.ClaimStateModel, 1)
		assert.EqualValues(t, newClaim.QualityAdjPower.String(), cp.ClaimStateModel[0].QualityAdjPower)
		assert.EqualValues(t, newClaim.RawBytePower.String(), cp.ClaimStateModel[0].RawBytePower)
	})
}

func TestPowerExtractV2(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.mustCreateEmptyPowerStateV2()
	minerAddr := tutils.NewIDAddr(t, 1234)

	t.Run("power state model", func(t *testing.T) {
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

		stateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: stateCid})

		info := actorstate.ActorInfo{
			Actor:           types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: stateCid},
			Address:         power.Address,
			TipSet:          stateTs,
			ParentStateRoot: stateTs.ParentState(),
			Epoch:           1,
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.EqualValues(t, info.ParentStateRoot.String(), cp.ChainPowerModel.StateRoot, "StateRoot")
		assert.EqualValues(t, state.TotalRawBytePower.String(), cp.ChainPowerModel.TotalRawBytesPower, "TotalRawBytesPower")
		assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.ChainPowerModel.TotalQABytesPower, "TotalQABytesPower")
		assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.ChainPowerModel.TotalRawBytesCommitted, "TotalRawBytesCommitted")
		assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.ChainPowerModel.TotalQABytesCommitted, "TotalQABytesCommitted")
		assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.ChainPowerModel.TotalPledgeCollateral, "TotalPledgeCollateral")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.ChainPowerModel.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.ChainPowerModel.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
		assert.EqualValues(t, state.MinerCount, cp.ChainPowerModel.MinerCount, "MinerCount")
		assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ChainPowerModel.ParticipatingMinerCount, "ParticipatingMinerCount")
	})

	t.Run("power claim model", func(t *testing.T) {
		claimMap, err := adt.AsMap(mapi.store, state.Claims)
		require.NoError(t, err)
		newClaim := power2.Claim{
			SealProofType:   abi.RegisteredSealProof_StackedDrg64GiBV1,
			RawBytePower:    abi.NewStoragePower(10),
			QualityAdjPower: abi.NewStoragePower(5),
		}

		oldStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		oldStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(oldStateTs.Key(), power.Address, &types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: oldStateCid})

		err = claimMap.Put(abi.AddrKey(minerAddr), &newClaim)
		require.NoError(t, err)

		state.Claims, err = claimMap.Root()
		require.NoError(t, err)

		newStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		newStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(newStateTs.Key(), power.Address, &types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: newStateCid})

		info := actorstate.ActorInfo{
			Epoch:           1,
			Actor:           types.Actor{Code: sa2builtin.StoragePowerActorCodeID, Head: newStateCid},
			Address:         power.Address,
			ParentTipSet:    oldStateTs,
			TipSet:          newStateTs,
			ParentStateRoot: newStateTs.ParentState(),
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.Len(t, cp.ClaimStateModel, 1)
		assert.EqualValues(t, newClaim.QualityAdjPower.String(), cp.ClaimStateModel[0].QualityAdjPower)
		assert.EqualValues(t, newClaim.RawBytePower.String(), cp.ClaimStateModel[0].RawBytePower)
	})
}

func TestPowerExtractV3(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.mustCreateEmptyPowerStateV3()
	minerAddr := tutils.NewIDAddr(t, 1234)

	t.Run("power state model", func(t *testing.T) {
		state.TotalRawBytePower = abi.NewStoragePower(1000)
		state.TotalBytesCommitted = abi.NewStoragePower(0)
		state.TotalQualityAdjPower = abi.NewStoragePower(0)
		state.TotalQABytesCommitted = abi.NewStoragePower(0)
		state.TotalPledgeCollateral = abi.NewTokenAmount(0)
		state.ThisEpochRawBytePower = abi.NewStoragePower(0)
		state.ThisEpochQualityAdjPower = abi.NewStoragePower(0)
		state.ThisEpochPledgeCollateral = abi.NewTokenAmount(0)
		state.ThisEpochQAPowerSmoothed = sa3smoothing.NewEstimate(big.NewInt(800_000_000_000), big.NewInt(6_000_000_000))
		state.MinerCount = 0
		state.MinerAboveMinPowerCount = 0

		stateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		stateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: stateCid})

		info := actorstate.ActorInfo{
			Actor:           types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: stateCid},
			Address:         power.Address,
			TipSet:          stateTs,
			ParentStateRoot: stateTs.ParentState(),
			Epoch:           1,
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.EqualValues(t, info.ParentStateRoot.String(), cp.ChainPowerModel.StateRoot, "StateRoot")
		assert.EqualValues(t, state.TotalRawBytePower.String(), cp.ChainPowerModel.TotalRawBytesPower, "TotalRawBytesPower")
		assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.ChainPowerModel.TotalQABytesPower, "TotalQABytesPower")
		assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.ChainPowerModel.TotalRawBytesCommitted, "TotalRawBytesCommitted")
		assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.ChainPowerModel.TotalQABytesCommitted, "TotalQABytesCommitted")
		assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.ChainPowerModel.TotalPledgeCollateral, "TotalPledgeCollateral")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.ChainPowerModel.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.ChainPowerModel.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
		assert.EqualValues(t, state.MinerCount, cp.ChainPowerModel.MinerCount, "MinerCount")
		assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ChainPowerModel.ParticipatingMinerCount, "ParticipatingMinerCount")
	})

	t.Run("power claim model", func(t *testing.T) {
		claimMap, err := adt3.AsMap(mapi.store, state.Claims, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)
		newClaim := power3.Claim{
			WindowPoStProofType: abi.RegisteredPoStProof_StackedDrgWinning64GiBV1,
			RawBytePower:        abi.NewStoragePower(10),
			QualityAdjPower:     abi.NewStoragePower(5),
		}

		oldStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		oldStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(oldStateTs.Key(), power.Address, &types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: oldStateCid})

		err = claimMap.Put(abi.AddrKey(minerAddr), &newClaim)
		require.NoError(t, err)

		state.Claims, err = claimMap.Root()
		require.NoError(t, err)

		newStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		newStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(newStateTs.Key(), power.Address, &types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: newStateCid})

		info := actorstate.ActorInfo{
			Epoch:           1,
			Actor:           types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: newStateCid},
			Address:         power.Address,
			ParentTipSet:    oldStateTs,
			TipSet:          newStateTs,
			ParentStateRoot: newStateTs.ParentState(),
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.Len(t, cp.ClaimStateModel, 1)
		assert.EqualValues(t, newClaim.QualityAdjPower.String(), cp.ClaimStateModel[0].QualityAdjPower)
		assert.EqualValues(t, newClaim.RawBytePower.String(), cp.ClaimStateModel[0].RawBytePower)
	})
}

func TestPowerExtractV4(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	state := mapi.mustCreateEmptyPowerStateV4()
	minerAddr := tutils.NewIDAddr(t, 1234)

	t.Run("power state model", func(t *testing.T) {
		state.TotalRawBytePower = abi.NewStoragePower(1000)
		state.TotalBytesCommitted = abi.NewStoragePower(0)
		state.TotalQualityAdjPower = abi.NewStoragePower(0)
		state.TotalQABytesCommitted = abi.NewStoragePower(0)
		state.TotalPledgeCollateral = abi.NewTokenAmount(0)
		state.ThisEpochRawBytePower = abi.NewStoragePower(0)
		state.ThisEpochQualityAdjPower = abi.NewStoragePower(0)
		state.ThisEpochPledgeCollateral = abi.NewTokenAmount(0)
		state.ThisEpochQAPowerSmoothed = sa4smoothing.NewEstimate(big.NewInt(800_000_000_000), big.NewInt(6_000_000_000))
		state.MinerCount = 0
		state.MinerAboveMinPowerCount = 0

		stateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		stateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(stateTs.Key(), power.Address, &types.Actor{Code: sa4builtin.StoragePowerActorCodeID, Head: stateCid})

		info := actorstate.ActorInfo{
			Actor:           types.Actor{Code: sa3builtin.StoragePowerActorCodeID, Head: stateCid},
			Address:         power.Address,
			TipSet:          stateTs,
			ParentStateRoot: stateTs.ParentState(),
			Epoch:           1,
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.EqualValues(t, info.ParentStateRoot.String(), cp.ChainPowerModel.StateRoot, "StateRoot")
		assert.EqualValues(t, state.TotalRawBytePower.String(), cp.ChainPowerModel.TotalRawBytesPower, "TotalRawBytesPower")
		assert.EqualValues(t, state.TotalQualityAdjPower.String(), cp.ChainPowerModel.TotalQABytesPower, "TotalQABytesPower")
		assert.EqualValues(t, state.TotalBytesCommitted.String(), cp.ChainPowerModel.TotalRawBytesCommitted, "TotalRawBytesCommitted")
		assert.EqualValues(t, state.TotalQABytesCommitted.String(), cp.ChainPowerModel.TotalQABytesCommitted, "TotalQABytesCommitted")
		assert.EqualValues(t, state.TotalPledgeCollateral.String(), cp.ChainPowerModel.TotalPledgeCollateral, "TotalPledgeCollateral")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.PositionEstimate.String(), cp.ChainPowerModel.QASmoothedPositionEstimate, "QASmoothedPositionEstimate")
		assert.EqualValues(t, state.ThisEpochQAPowerSmoothed.VelocityEstimate.String(), cp.ChainPowerModel.QASmoothedVelocityEstimate, "QASmoothedVelocityEstimate")
		assert.EqualValues(t, state.MinerCount, cp.ChainPowerModel.MinerCount, "MinerCount")
		assert.EqualValues(t, state.MinerAboveMinPowerCount, cp.ChainPowerModel.ParticipatingMinerCount, "ParticipatingMinerCount")
	})

	t.Run("power claim model", func(t *testing.T) {
		claimMap, err := adt3.AsMap(mapi.store, state.Claims, sa3builtin.DefaultHamtBitwidth)
		require.NoError(t, err)
		newClaim := power3.Claim{
			WindowPoStProofType: abi.RegisteredPoStProof_StackedDrgWinning64GiBV1,
			RawBytePower:        abi.NewStoragePower(10),
			QualityAdjPower:     abi.NewStoragePower(5),
		}

		oldStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		oldStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(oldStateTs.Key(), power.Address, &types.Actor{Code: sa4builtin.StoragePowerActorCodeID, Head: oldStateCid})

		err = claimMap.Put(abi.AddrKey(minerAddr), &newClaim)
		require.NoError(t, err)

		state.Claims, err = claimMap.Root()
		require.NoError(t, err)

		newStateCid, err := mapi.Store().Put(ctx, state)
		require.NoError(t, err)

		newStateTs := mapi.fakeTipset(minerAddr, 1)
		mapi.setActor(newStateTs.Key(), power.Address, &types.Actor{Code: sa4builtin.StoragePowerActorCodeID, Head: newStateCid})

		info := actorstate.ActorInfo{
			Epoch:           1,
			Actor:           types.Actor{Code: sa4builtin.StoragePowerActorCodeID, Head: newStateCid},
			Address:         power.Address,
			ParentTipSet:    oldStateTs,
			TipSet:          newStateTs,
			ParentStateRoot: newStateTs.ParentState(),
		}

		ex := actorstate.StoragePowerExtractor{}
		res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
		require.NoError(t, err)

		cp, ok := res.(*powermodel.PowerTaskResult)
		require.True(t, ok)
		require.NotNil(t, cp)

		assert.Len(t, cp.ClaimStateModel, 1)
		assert.EqualValues(t, newClaim.QualityAdjPower.String(), cp.ClaimStateModel[0].QualityAdjPower)
		assert.EqualValues(t, newClaim.RawBytePower.String(), cp.ClaimStateModel[0].RawBytePower)
	})
}
