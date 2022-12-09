package minerdiff

import (
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
)

var MinerStateLoader = func(store adt.Store, act *types.Actor) (interface{}, error) {
	return miner.Load(store, act)
}
var MinerPreCommitMapLoader = func(m interface{}) (adt.Map, *adt.MapOpts, error) {
	minerState := m.(miner.State)
	perCommitMap, err := minerState.PrecommitsMap()
	if err != nil {
		return nil, nil, err
	}
	return perCommitMap, &adt.MapOpts{
		Bitwidth: minerState.PrecommitsMapBitWidth(),
		HashFunc: minerState.PrecommitsMapHashFunction(),
	}, nil
}

var MinerSectorArrayLoader = func(m interface{}) (adt.Array, int, error) {
	minerState := m.(miner.State)
	sectorArray, err := minerState.SectorsArray()
	if err != nil {
		return nil, -1, err
	}
	return sectorArray, minerState.SectorsAmtBitwidth(), nil
}
