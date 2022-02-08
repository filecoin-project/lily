package actorstate_test

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lotus/chain/types"

	sabuiltin "github.com/filecoin-project/specs-actors/actors/builtin"
	samarket "github.com/filecoin-project/specs-actors/actors/builtin/market"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/testutil"
)

type balance struct {
	available abi.TokenAmount
	locked    abi.TokenAmount
}

func TestMarketPredicates(t *testing.T) {
	ctx := context.Background()

	mapi := NewMockAPI(t)

	oldDeal1 := &samarket.DealState{
		SectorStartEpoch: 1,
		LastUpdatedEpoch: 2,
		SlashEpoch:       0,
	}
	oldDeal2 := &samarket.DealState{
		SectorStartEpoch: 4,
		LastUpdatedEpoch: 5,
		SlashEpoch:       0,
	}
	oldDeals := map[abi.DealID]*samarket.DealState{
		abi.DealID(1): oldDeal1,
		abi.DealID(2): oldDeal2,
	}

	oldProp1 := &samarket.DealProposal{
		PieceCID:             testutil.RandomCid(),
		PieceSize:            0,
		VerifiedDeal:         false,
		Client:               tutils.NewIDAddr(t, 1),
		Provider:             tutils.NewIDAddr(t, 1),
		StartEpoch:           1,
		EndEpoch:             2,
		StoragePricePerEpoch: big.Zero(),
		ProviderCollateral:   big.Zero(),
		ClientCollateral:     big.Zero(),
	}
	oldProp2 := &samarket.DealProposal{
		PieceCID:             testutil.RandomCid(),
		PieceSize:            0,
		VerifiedDeal:         false,
		Client:               tutils.NewIDAddr(t, 1),
		Provider:             tutils.NewIDAddr(t, 1),
		StartEpoch:           2,
		EndEpoch:             3,
		StoragePricePerEpoch: big.Zero(),
		ProviderCollateral:   big.Zero(),
		ClientCollateral:     big.Zero(),
	}
	oldProps := map[abi.DealID]*samarket.DealProposal{
		abi.DealID(1): oldProp1,
		abi.DealID(2): oldProp2,
	}

	oldBalances := map[address.Address]balance{
		tutils.NewIDAddr(t, 1): {abi.NewTokenAmount(1000), abi.NewTokenAmount(1000)},
		tutils.NewIDAddr(t, 2): {abi.NewTokenAmount(2000), abi.NewTokenAmount(500)},
		tutils.NewIDAddr(t, 3): {abi.NewTokenAmount(3000), abi.NewTokenAmount(2000)},
		tutils.NewIDAddr(t, 5): {abi.NewTokenAmount(3000), abi.NewTokenAmount(1000)},
	}

	oldStateCid := mapi.mustCreateMarketState(ctx, oldDeals, oldProps, oldBalances)

	newDeal1 := &samarket.DealState{
		SectorStartEpoch: 1,
		LastUpdatedEpoch: 3,
		SlashEpoch:       0,
	}

	// deal 2 removed

	// added
	newDeal3 := &samarket.DealState{
		SectorStartEpoch: 1,
		LastUpdatedEpoch: 2,
		SlashEpoch:       3,
	}
	newDeals := map[abi.DealID]*samarket.DealState{
		abi.DealID(1): newDeal1,
		// deal 2 was removed
		abi.DealID(3): newDeal3,
	}

	// added
	newProp3 := &samarket.DealProposal{
		PieceCID:             testutil.RandomCid(),
		PieceSize:            0,
		VerifiedDeal:         false,
		Client:               tutils.NewIDAddr(t, 1),
		Provider:             tutils.NewIDAddr(t, 1),
		StartEpoch:           4,
		EndEpoch:             4,
		StoragePricePerEpoch: big.Zero(),
		ProviderCollateral:   big.Zero(),
		ClientCollateral:     big.Zero(),
	}
	newProps := map[abi.DealID]*samarket.DealProposal{
		abi.DealID(1): oldProp1, // 1 was persisted
		// prop 2 was removed
		abi.DealID(3): newProp3, // new
	}
	newBalances := map[address.Address]balance{
		tutils.NewIDAddr(t, 1): {abi.NewTokenAmount(3000), abi.NewTokenAmount(0)},
		tutils.NewIDAddr(t, 2): {abi.NewTokenAmount(2000), abi.NewTokenAmount(500)},
		tutils.NewIDAddr(t, 4): {abi.NewTokenAmount(5000), abi.NewTokenAmount(0)},
		tutils.NewIDAddr(t, 5): {abi.NewTokenAmount(1000), abi.NewTokenAmount(3000)},
	}

	newStateCid := mapi.mustCreateMarketState(ctx, newDeals, newProps, newBalances)

	minerAddr := tutils.NewIDAddr(t, 0o0)

	oldStateTs := mapi.fakeTipset(minerAddr, 1)
	mapi.setActor(oldStateTs.Key(), market.Address, &types.Actor{Code: sabuiltin.StorageMarketActorCodeID, Head: oldStateCid})
	newStateTs := mapi.fakeTipset(minerAddr, 2)
	mapi.setActor(newStateTs.Key(), market.Address, &types.Actor{Code: sabuiltin.StorageMarketActorCodeID, Head: newStateCid})

	info := actorstate.ActorInfo{
		Actor:        types.Actor{Code: sabuiltin.StorageMarketActorCodeID, Head: newStateCid},
		Address:      market.Address,
		TipSet:       newStateTs,
		ParentTipSet: oldStateTs,
		Epoch:        1, // must be greater than zero else this has special handling for genesis block.
	}

	ex := actorstate.StorageMarketExtractor{}
	res, err := ex.Extract(ctx, info, []*lens.ExecutedMessage{}, mapi)
	require.NoError(t, err)

	mtr, ok := res.(*marketmodel.MarketTaskResult)
	require.True(t, ok)
	require.NotNil(t, mtr)

	t.Run("proposals", func(t *testing.T) {
		require.Equal(t, 1, len(mtr.Proposals))

		assert.EqualValues(t, abi.DealID(3), mtr.Proposals[0].DealID, "DealID")
		assert.EqualValues(t, newStateTs.ParentState().String(), mtr.Proposals[0].StateRoot, "StateRoot")
		assert.EqualValues(t, newProp3.PieceSize, mtr.Proposals[0].PaddedPieceSize, "PaddedPieceSize")
		assert.EqualValues(t, newProp3.PieceSize.Unpadded(), mtr.Proposals[0].UnpaddedPieceSize, "UnpaddedPieceSize")
		assert.EqualValues(t, newProp3.StartEpoch, mtr.Proposals[0].StartEpoch, "StartEpoch")
		assert.EqualValues(t, newProp3.EndEpoch, mtr.Proposals[0].EndEpoch, "EndEpoch")
		assert.EqualValues(t, newProp3.Client.String(), mtr.Proposals[0].ClientID, "ClientID")
		assert.EqualValues(t, newProp3.Provider.String(), mtr.Proposals[0].ProviderID, "ProviderID")
		assert.EqualValues(t, newProp3.ClientCollateral.String(), mtr.Proposals[0].ClientCollateral, "ClientCollateral")
		assert.EqualValues(t, newProp3.ProviderCollateral.String(), mtr.Proposals[0].ProviderCollateral, "ProviderCollateral")
		assert.EqualValues(t, newProp3.StoragePricePerEpoch.String(), mtr.Proposals[0].StoragePricePerEpoch, "StoragePricePerEpoch")
		assert.EqualValues(t, newProp3.PieceCID.String(), mtr.Proposals[0].PieceCID, "PieceCID")
		assert.EqualValues(t, newProp3.VerifiedDeal, mtr.Proposals[0].IsVerified, "IsVerified")
		assert.EqualValues(t, newProp3.Label, mtr.Proposals[0].Label, "Label")
	})

	t.Run("states", func(t *testing.T) {
		require.Equal(t, 2, len(mtr.States))

		assert.EqualValues(t, abi.DealID(3), mtr.States[0].DealID, "DealID")
		assert.EqualValues(t, newDeal3.SectorStartEpoch, mtr.States[0].SectorStartEpoch, "SectorStartEpoch")
		assert.EqualValues(t, newDeal3.LastUpdatedEpoch, mtr.States[0].LastUpdateEpoch, "LastUpdateEpoch")
		assert.EqualValues(t, newDeal3.SlashEpoch, mtr.States[0].SlashEpoch, "SlashEpoch")
		assert.EqualValues(t, newStateTs.ParentState().String(), mtr.States[0].StateRoot, "StateRoot")

		assert.EqualValues(t, abi.DealID(1), mtr.States[1].DealID, "DealID")
		assert.EqualValues(t, newDeal1.SectorStartEpoch, mtr.States[1].SectorStartEpoch, "SectorStartEpoch")
		assert.EqualValues(t, newDeal1.LastUpdatedEpoch, mtr.States[1].LastUpdateEpoch, "LastUpdateEpoch")
		assert.EqualValues(t, newDeal1.SlashEpoch, mtr.States[1].SlashEpoch, "SlashEpoch")
		assert.EqualValues(t, newStateTs.ParentState().String(), mtr.States[1].StateRoot, "StateRoot")
	})
}

func TestSanitizeLabel(t *testing.T) {
	testCases := []struct {
		in   string
		want string
	}{
		{
			in:   "this is ok",
			want: "this is ok",
		},

		// Example from https://github.com/filecoin-project/lily/issues/843
		{
			in: string([]byte{
				0xa1, 0x65, 0x70, 0x63, 0x69, 0x64,
				0x73, 0x81, 0xd8, 0x2a, 0x58, 0x23,
				0x00, 0x12, 0x20, 0x7a, 0x95, 0xa5,
				0x59, 0x2a, 0xac, 0x47, 0x35, 0x0e,
				0x80, 0x5b, 0x09, 0xe4, 0xba, 0xbd,
				0x16, 0xf0, 0xff, 0xee, 0x24, 0xc7,
				0x17, 0x00, 0x7a, 0x58, 0x8c, 0x22,
				0x22, 0x7c, 0x7d, 0xe5, 0xd6, 0x9d,
				0x22,
			}),
			want: "\ufffdepcids\ufffd\ufffd*X#\x00\x12 z\ufffd\ufffdY*\ufffdG5\x0e\ufffd[\t\u4ebd\x16\ufffd\ufffd\ufffd$\ufffd\x17\x00zX\ufffd\"\"|}\ufffd\u059d\"",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			got := actorstate.SanitizeLabel(tc.in)

			if got != tc.want {
				t.Errorf("got %+q, wanted %q", got, tc.want)
			}
		})
	}
}
