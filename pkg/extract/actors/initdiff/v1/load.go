package v1

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
)

var InitStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return init_.Load(store, act)
}

var InitAddressesMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	initState := m.(init_.State)
	addressesMap, err := initState.AddressMap()
	if err != nil {
		return nil, nil, err
	}
	return addressesMap, &adt.MapOpts{
		Bitwidth: initState.AddressMapBitWidth(),
		HashFunc: initState.AddressMapHashFunction(),
	}, nil
}
