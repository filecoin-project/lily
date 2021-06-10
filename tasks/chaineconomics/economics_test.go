package chaineconomics

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/sentinel-visor/testutil"
)

type MockedChainEconomicsLens struct {
	mock.Mock
}

func (m *MockedChainEconomicsLens) StateVMCirculatingSupplyInternal(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(api.CirculatingSupply), args.Error(1)
}

func TestEconomicsModelExtraction(t *testing.T) {
	ctx := context.Background()

	expectedTs := testutil.FakeTipset(t)
	expectedCircSupply := api.CirculatingSupply{
		FilVested:           abi.NewTokenAmount(1),
		FilMined:            abi.NewTokenAmount(2),
		FilBurnt:            abi.NewTokenAmount(3),
		FilLocked:           abi.NewTokenAmount(4),
		FilCirculating:      abi.NewTokenAmount(5),
		FilReserveDisbursed: abi.NewTokenAmount(6),
	}

	mockedLens := new(MockedChainEconomicsLens)
	mockedLens.On("StateVMCirculatingSupplyInternal", mock.Anything, expectedTs.Key()).Return(expectedCircSupply, nil)

	em, err := ExtractChainEconomicsModel(ctx, mockedLens, expectedTs)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedTs.ParentState().String(), em.ParentStateRoot)
	assert.EqualValues(t, expectedCircSupply.FilBurnt.String(), em.BurntFil)
	assert.EqualValues(t, expectedCircSupply.FilMined.String(), em.MinedFil)
	assert.EqualValues(t, expectedCircSupply.FilVested.String(), em.VestedFil)
	assert.EqualValues(t, expectedCircSupply.FilLocked.String(), em.LockedFil)
	assert.EqualValues(t, expectedCircSupply.FilCirculating.String(), em.CirculatingFil)
	assert.EqualValues(t, expectedCircSupply.FilReserveDisbursed.String(), em.FilReserveDisbursed)
}
