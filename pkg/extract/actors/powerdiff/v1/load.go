package v1

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
)

var PowerStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return power.Load(store, act)
}

var PowerClaimsMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	powerState := m.(power.State)
	claimsMap, err := powerState.ClaimsMap()
	if err != nil {
		return nil, nil, err
	}
	return claimsMap, &adt.MapOpts{
		Bitwidth: powerState.ClaimsMapBitWidth(),
		HashFunc: powerState.ClaimsMapHashFunction(),
	}, nil
}
