package generic

import (
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/actors/adt"
)

var log = logging.Logger("lily/extract/diff")

// TODO use go-state-types store

type ActorStateArrayLoader = func(interface{}) (adt.Array, int, error)
type ActorStateLoader = func(adt.Store, *types.Actor) (interface{}, error)
type ActorStateMapLoader = func(interface{}) (adt.Map, *adt.MapOpts, error)
