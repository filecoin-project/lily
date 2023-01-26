package verifreg

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	v1 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v2"
)

func TransformVerifregState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid) (model.Persistable, error) {
	switch version {
	case actortypes.Version0,
		actortypes.Version2,
		actortypes.Version3,
		actortypes.Version4,
		actortypes.Version5,
		actortypes.Version6,
		actortypes.Version7,
		actortypes.Version8:
		return v1.TransformVerifregState(ctx, s, version, current, executed, root)

	case actortypes.Version9:
		return v2.TransformVerifregState(ctx, s, version, current, executed, root)
	}
	return nil, fmt.Errorf("unsupported version : %d", version)

}
