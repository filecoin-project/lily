package init_

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/store"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, s store.Store, addresses actors.ActorDiffResult) (cid.Cid, error) {
	isc, err := addresses.MarshalStateChange(ctx, s)
	if err != nil {
		return cid.Undef, err
	}
	return s.Put(ctx, isc)
}
