package v9

import (
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
)

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
