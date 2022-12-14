package cbor

import (
	"context"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/extract/procesor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/miner"
)

type ActorIPLDContainer struct {
	// HAMT of actor addresses to their changes
	// HAMT of miner address to some big structure...
}

func ProcessActors(ctx context.Context, store adt.Store, changes *procesor.ActorStateChanges) (interface{}, error) {
	m, err := adt.MakeEmptyMap(store, 5 /*TODO*/)
	if err != nil {
		return nil, err
	}
	if err := miner.HandleChanges(ctx, store, m, changes.MinerActors); err != nil {
		return nil, err
	}
	return nil, nil
}
