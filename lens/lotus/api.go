package lotus

import (
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
)

func NewAPIWrapper(node api.FullNode, store adt.Store) *APIWrapper {
	return &APIWrapper{
		FullNode: node,
		store:    store,
	}
}

var _ lens.API = &APIWrapper{}

type APIWrapper struct {
	api.FullNode
	store adt.Store
}

func (aw *APIWrapper) Store() adt.Store {
	return aw.store
}
