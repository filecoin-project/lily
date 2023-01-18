package miner

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, s store.Store, miners map[address.Address]actors.ActorDiffResult) (cid.Cid, error) {
	minerHamt, err := adt.MakeEmptyMap(s, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for addr, change := range miners {
		msc, err := change.MarshalStateChange(ctx, s)
		if err != nil {
			return cid.Undef, err
		}

		if err := minerHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return cid.Undef, err
		}
	}
	return minerHamt.Root()
}
