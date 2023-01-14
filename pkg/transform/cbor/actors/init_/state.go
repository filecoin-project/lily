package init_

import (
	"context"

	"github.com/filecoin-project/lotus/blockstore"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, bs blockstore.Blockstore, addresses actors.ActorDiffResult) (cid.Cid, error) {
	store := adt2.WrapBlockStore(ctx, bs)
	isc, err := addresses.MarshalStateChange(ctx, bs)
	if err != nil {
		return cid.Undef, err
	}
	return store.Put(ctx, isc)
}
