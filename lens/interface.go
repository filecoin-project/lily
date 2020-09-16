package lens

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
)

type API interface {
	Store() adt.Store
	api.FullNode
}
