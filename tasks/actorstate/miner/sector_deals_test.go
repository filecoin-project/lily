package miner_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	minerstate "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	minerstatemocks "github.com/filecoin-project/lily/chain/actors/builtin/miner/mocks"
	"github.com/filecoin-project/lily/tasks/actorstate/miner"
	extmocks "github.com/filecoin-project/lily/tasks/actorstate/miner/extraction/mocks"
	atesting "github.com/filecoin-project/lily/tasks/test"
	"github.com/filecoin-project/lily/testutil"
)

func TestExtractSectorDealsModelWithParentState(t *testing.T) {
	ctx := context.Background()

	currentTipSet := testutil.MustFakeTipSet(t, 10)
	currentMinerState := new(minerstatemocks.State)

	parentTipSet := testutil.MustFakeTipSet(t, 9)
	parentMinerState := new(minerstatemocks.State)

	minerAddr := testutil.MustMakeAddress(t, 111)

	mActorStateAPI := new(atesting.MockActorStateAPI)

	mExtractionState := new(extmocks.MockMinerState)
	mExtractionState.On("Address").Return(minerAddr)
	mExtractionState.On("CurrentTipSet").Return(currentTipSet)
	mExtractionState.On("ParentTipSet").Return(parentTipSet)
	mExtractionState.On("CurrentState").Return(currentMinerState)
	mExtractionState.On("ParentState").Return(parentMinerState)
	mExtractionState.On("API").Return(mActorStateAPI)
	mActorStateAPI.On("DiffSectors", mock.Anything, minerAddr, currentTipSet, parentTipSet, parentMinerState, currentMinerState).
		Return(&minerstate.SectorChanges{
			Added: []minerstate.SectorOnChainInfo{
				{
					DeprecatedDealIDs: []abi.DealID{1},
					SectorNumber:      1,
				},
				{
					DeprecatedDealIDs: []abi.DealID{2},
					SectorNumber:      2,
				},
				{
					DeprecatedDealIDs: []abi.DealID{3},
					SectorNumber:      3,
				},
			},
			Snapped: []minerstate.SectorModification{
				{
					To: minerstate.SectorOnChainInfo{
						DeprecatedDealIDs: []abi.DealID{4},
						SectorNumber:      4,
					},
				},
				{
					To: minerstate.SectorOnChainInfo{
						DeprecatedDealIDs: []abi.DealID{5},
						SectorNumber:      5,
					},
				},
				{
					To: minerstate.SectorOnChainInfo{
						DeprecatedDealIDs: []abi.DealID{6},
						SectorNumber:      6,
					},
				},
			},
		}, nil)
	result, err := miner.ExtractSectorDealsModel(ctx, mExtractionState)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 6)

	sectorDeals := make(map[uint64]uint64)
	for _, res := range result {
		sectorDeals[res.SectorID] = res.DealID
		require.Equal(t, int64(currentTipSet.Height()), res.Height)
		require.Equal(t, minerAddr.String(), res.MinerID)
	}
	require.Len(t, sectorDeals, 6)
	for sector, deal := range sectorDeals {
		require.Equal(t, sector, deal)
	}
}

func TestExtractSectorDealsModelNoParentState(t *testing.T) {
	ctx := context.Background()

	currentTipSet := testutil.MustFakeTipSet(t, 10)
	currentMinerState := new(minerstatemocks.State)

	parentTipSet := testutil.MustFakeTipSet(t, 9)

	minerAddr := testutil.MustMakeAddress(t, 111)

	mExtractionState := new(extmocks.MockMinerState)
	mExtractionState.On("Address").Return(minerAddr)
	mExtractionState.On("CurrentTipSet").Return(currentTipSet)
	mExtractionState.On("ParentTipSet").Return(parentTipSet)
	mExtractionState.On("CurrentState").Return(currentMinerState)
	mExtractionState.On("ParentState").Return(nil)
	var bf *bitfield.BitField
	currentMinerState.On("LoadSectors", bf).Return([]*minerstate.SectorOnChainInfo{
		{
			SectorNumber:      1,
			DeprecatedDealIDs: []abi.DealID{1},
		},
		{
			DeprecatedDealIDs: []abi.DealID{2},
			SectorNumber:      2,
		},
		{
			DeprecatedDealIDs: []abi.DealID{3},
			SectorNumber:      3,
		},
		{
			DeprecatedDealIDs: []abi.DealID{4},
			SectorNumber:      4,
		},
		{
			DeprecatedDealIDs: []abi.DealID{5},
			SectorNumber:      5,
		},
		{
			DeprecatedDealIDs: []abi.DealID{6},
			SectorNumber:      6,
		},
	}, nil)

	result, err := miner.ExtractSectorDealsModel(ctx, mExtractionState)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 6)

	sectorDeals := make(map[uint64]uint64)
	for _, res := range result {
		sectorDeals[res.SectorID] = res.DealID
		require.Equal(t, int64(currentTipSet.Height()), res.Height)
		require.Equal(t, minerAddr.String(), res.MinerID)
	}
	require.Len(t, sectorDeals, 6)
	for sector, deal := range sectorDeals {
		require.Equal(t, sector, deal)
	}
}
