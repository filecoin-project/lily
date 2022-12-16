package miner

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/lotus/blockstore"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, bs blockstore.Blockstore, miners map[address.Address]actors.ActorDiffResult) (cid.Cid, error) {
	store := adt2.WrapBlockStore(ctx, bs)
	minerHamt, err := adt.MakeEmptyMap(store, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for addr, change := range miners {
		msc, err := change.MarshalStateChange(ctx, bs)
		if err != nil {
			return cid.Undef, err
		}

		if err := minerHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return cid.Undef, err
		}
	}
	return minerHamt.Root()
}
