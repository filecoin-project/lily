package market

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChange(ctx context.Context, s store.Store, market actors.ActorDiffResult) (cid.Cid, error) {
	vsc, err := market.MarshalStateChange(ctx, s)
	if err != nil {
		return cid.Undef, err
	}
	return s.Put(ctx, vsc)

}
