package chaineconomics

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/testutil"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

// nolint: revive
type MockedChainEconomicsLens struct {
	mock.Mock
}

func (m *MockedChainEconomicsLens) CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error) {
	args := m.Called(ctx, ts)
	return args.Get(0).(api.CirculatingSupply), args.Error(1)
}

func (m *MockedChainEconomicsLens) Actor(_ context.Context, _ address.Address, _ types.TipSetKey) (*types.Actor, error) {
	return nil, nil
}
func (m *MockedChainEconomicsLens) Store() adt.Store {
	return nil
}
func (m *MockedChainEconomicsLens) MinerLoad(_ adt.Store, _ *types.Actor) (miner.State, error) {
	return nil, fmt.Errorf("test error")
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
	mockedLens.On("CirculatingSupply", mock.Anything, expectedTs).Return(expectedCircSupply, nil)

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
