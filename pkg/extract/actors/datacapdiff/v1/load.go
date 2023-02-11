package v1

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/datacap"
)

var DatacapStateLoader = func(s adt.Store, act *types.Actor) (interface{}, error) {
	return datacap.Load(s, act)
}

var DatacapBalancesMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	datacapState := m.(datacap.State)
	balanceMap, err := datacapState.VerifiedClients()
	if err != nil {
		return nil, nil, err
	}
	return balanceMap, &adt.MapOpts{
		Bitwidth: datacapState.VerifiedClientsMapBitWidth(),
		HashFunc: datacapState.VerifiedClientsMapHashFunction(),
	}, nil
}

var DatacapAllowancesMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	datacapState := m.(datacap.State)
	allowanceMap, err := datacapState.AllowanceMap()
	if err != nil {
		return nil, nil, err
	}
	return allowanceMap, &adt.MapOpts{
		Bitwidth: datacapState.AllowanceMapBitWidth(),
		HashFunc: datacapState.VerifiedClientsMapHashFunction(),
	}, nil

}
