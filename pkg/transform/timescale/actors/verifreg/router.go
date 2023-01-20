package verifreg

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v0"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v9"
)

type VerifregHandler = func(ctx context.Context, s store.Store, current, executed *types.TipSet, verifregRoot cid.Cid) (model.PersistableList, error)

func MakeVerifregProcessor(av actortypes.Version) (VerifregHandler, error) {
	switch av {
	case actortypes.Version0:
		return v0.VerifregHandler, nil
	case actortypes.Version9:
		return v9.VerifregHandler, nil
	default:
		return nil, fmt.Errorf("unsupported verifreg actor version: %d", av)
	}
}
