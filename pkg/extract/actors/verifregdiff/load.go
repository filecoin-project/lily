package verifregdiff

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
)

// VerifregStateLoader returns the verifiergistry actor state for `act`.
var VerifregStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return verifreg.Load(store, act)
}

var VerifiregVerifiersMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	verifregState := m.(verifreg.State)
	verifierMap, err := verifregState.VerifiersMap()
	if err != nil {
		return nil, nil, err
	}
	return verifierMap, &adt.MapOpts{
		Bitwidth: verifregState.VerifiersMapBitWidth(),
		HashFunc: verifregState.VerifiersMapHashFunction(),
	}, nil
}

var VerifiregClientsMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	verifregState := m.(verifreg.State)
	clientsMap, err := verifregState.VerifiedClientsMap()
	if err != nil {
		return nil, nil, err
	}
	return clientsMap, &adt.MapOpts{
		Bitwidth: verifregState.VerifiedClientsMapBitWidth(),
		HashFunc: verifregState.VerifiedClientsMapHashFunction(),
	}, nil
}

var VerifiiregClaimsMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	verifregState := m.(verifreg.State)
	claimsMap, err := verifregState.ClaimsMap()
	if err != nil {
		return nil, nil, err
	}
	return claimsMap, &adt.MapOpts{
		Bitwidth: verifregState.ClaimsMapBitWidth(),
		HashFunc: verifregState.ClaimsMapHashFunction(),
	}, nil
}
