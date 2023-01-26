package miner

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	v1 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1"
	v1_0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v0"
	v1_2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v2"
	v1_3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v3"
	v1_4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v4"
	v1_5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v5"
	v1_6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v6"
	v1_7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v7"
	v1_8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v8"
	v1_9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v9"
)

func HandleMiner(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *rawdiff.ActorChange, version actortypes.Version) (model.Persistable, error) {
	switch version {
	case actortypes.Version0:
		return v1_0.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version2:
		return v1_2.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version3:
		return v1_3.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version4:
		return v1_4.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version5:
		return v1_5.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version6:
		return v1_6.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version7:
		return v1_7.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version8:
		return v1_8.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version9:
		return v1_9.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version10:
		panic("not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported miner actor version: %d", version)
	}
}

func TransformMinerState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid) (model.Persistable, error) {
	switch version {
	case actortypes.Version0,
		actortypes.Version2,
		actortypes.Version3,
		actortypes.Version4,
		actortypes.Version5,
		actortypes.Version6,
		actortypes.Version7,
		actortypes.Version8,
		actortypes.Version9:
		return v1.TransformMinerStates(ctx, s, version, current, executed, root)
	}
	return nil, fmt.Errorf("unsupported version : %d", version)

}
