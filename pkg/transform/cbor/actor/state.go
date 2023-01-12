package actor

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	store2 "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, bs blockstore.Blockstore, actors map[address.Address]actors.ActorDiffResult) (cid.Cid, error) {
	store := store2.WrapBlockStore(ctx, bs)
	actorHamt, err := adt.MakeEmptyMap(store, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for addr, change := range actors {
		msc, err := change.MarshalStateChange(ctx, bs)
		if err != nil {
			return cid.Undef, err
		}

		if err := actorHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return cid.Undef, err
		}
	}
	return actorHamt.Root()
}
