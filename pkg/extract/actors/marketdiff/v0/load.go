package v0

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
)

var MarketStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return market.Load(store, act)
}

var MarketDealsArrayLoader = func(m interface{}) (adt.Array, int, error) {
	marketState := m.(market.State)
	dealsArray, err := marketState.States()
	if err != nil {
		return nil, -1, err
	}
	return dealsArray.AsArray(), marketState.DealStatesAmtBitwidth(), nil
}

var MarketProposlasArrayLoader = func(m interface{}) (adt.Array, int, error) {
	marketState := m.(market.State)
	proposalsArray, err := marketState.Proposals()
	if err != nil {
		return nil, -1, err
	}
	return proposalsArray.AsArray(), marketState.DealProposalsAmtBitwidth(), nil
}
