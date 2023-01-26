package init

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	initdiff "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v1"
)

func InitHandler(ctx context.Context, s store.Store, current, executed *types.TipSet, initMapRoot cid.Cid) (model.Persistable, error) {
	initState := new(initdiff.StateChange)
	if err := s.Get(ctx, initMapRoot, initState); err != nil {
		return nil, err
	}

	initStateDiff, err := initState.ToStateDiffResult(ctx, s)
	if err != nil {
		return nil, err
	}

	return Addresses{}.Extract(ctx, current, executed, initStateDiff)
}
