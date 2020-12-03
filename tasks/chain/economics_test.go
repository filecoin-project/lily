package chain

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/sentinel-visor/testutil"
)

type MockedEconomicsLens struct {
	mock.Mock
}

func (m *MockedEconomicsLens) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(*types.TipSet), args.Error(1)
}

func (m *MockedEconomicsLens) StateVMCirculatingSupplyInternal(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(api.CirculatingSupply), args.Error(1)
}

func fakeTipset(t testing.TB) *types.TipSet {
	bh := &types.BlockHeader{
		Miner:                 tutils.NewIDAddr(t, 123),
		Height:                1,
		ParentStateRoot:       testutil.RandomCid(),
		Messages:              testutil.RandomCid(),
		ParentMessageReceipts: testutil.RandomCid(),
		BlockSig:              &crypto.Signature{Type: crypto.SigTypeBLS},
		BLSAggregate:          &crypto.Signature{Type: crypto.SigTypeBLS},
		Timestamp:             0,
	}
	ts, err := types.NewTipSet([]*types.BlockHeader{bh})
	if err != nil {
		panic(err)
	}
	return ts
}

func TestEconomicsModelExtraction(t *testing.T) {
	ctx := context.Background()

	expectedTs := fakeTipset(t)
	expectedCircSupply := api.CirculatingSupply{
		FilVested:      abi.NewTokenAmount(1),
		FilMined:       abi.NewTokenAmount(2),
		FilBurnt:       abi.NewTokenAmount(3),
		FilLocked:      abi.NewTokenAmount(4),
		FilCirculating: abi.NewTokenAmount(5),
	}

	mockedLens := new(MockedEconomicsLens)
	mockedLens.On("ChainGetTipSet", ctx, expectedTs.Key()).Return(expectedTs, nil)
	mockedLens.On("StateVMCirculatingSupplyInternal", ctx, expectedTs.Key()).Return(expectedCircSupply, nil)

	em, err := extractChainEconomicsModel(ctx, mockedLens, expectedTs.Key())
	assert.NoError(t, err)
	assert.EqualValues(t, expectedTs.ParentState().String(), em.ParentStateRoot)
	assert.EqualValues(t, expectedCircSupply.FilBurnt.String(), em.BurntFil)
	assert.EqualValues(t, expectedCircSupply.FilMined.String(), em.MinedFil)
	assert.EqualValues(t, expectedCircSupply.FilVested.String(), em.VestedFil)
	assert.EqualValues(t, expectedCircSupply.FilLocked.String(), em.LockedFil)
	assert.EqualValues(t, expectedCircSupply.FilCirculating.String(), em.CirculatingFil)
}
