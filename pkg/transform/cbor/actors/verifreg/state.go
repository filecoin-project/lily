package verifreg

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func HandleChanges(ctx context.Context, s store.Store, verifreg actors.ActorDiffResult) (cid.Cid, error) {
	vsc, err := verifreg.MarshalStateChange(ctx, s)
	if err != nil {
		return cid.Undef, err
	}
	return s.Put(ctx, vsc)
}
