package verifreg

import (
	"context"

	"github.com/filecoin-project/lotus/blockstore"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, bs blockstore.Blockstore, verifreg actors.ActorDiffResult) (cid.Cid, error) {
	store := adt2.WrapBlockStore(ctx, bs)
	vsc, err := verifreg.MarshalStateChange(ctx, bs)
	if err != nil {
		return cid.Undef, err
	}
	return store.Put(ctx, vsc)
}
