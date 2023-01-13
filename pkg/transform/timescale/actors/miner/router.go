package miner

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v0"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v2"
	v3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v3"
	v4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v4"
	v5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v5"
	v6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v6"
	v7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v7"
	v8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v8"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v9"
)

func HandleMiner(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange, version actortypes.Version) (model.Persistable, error) {
	switch version {
	case actortypes.Version0:
		return v0.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version2:
		return v2.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version3:
		return v3.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version4:
		return v4.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version5:
		return v5.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version6:
		return v6.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version7:
		return v7.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version8:
		return v8.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version9:
		return v9.MinerStateHandler(ctx, current, executed, addr, change)
	case actortypes.Version10:
		panic("not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported miner actor version: %d", version)
	}
}

type MinerHandler = func(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, minerMapRoot cid.Cid) (model.PersistableList, error)

func MakeMinerProcessor(av actortypes.Version) (MinerHandler, error) {
	switch av {
	case actortypes.Version0:
		return v0.MinerHandler, nil
	case actortypes.Version2:
		return v2.MinerHandler, nil
	case actortypes.Version3:
		return v3.MinerHandler, nil
	case actortypes.Version4:
		return v4.MinerHandler, nil
	case actortypes.Version5:
		return v5.MinerHandler, nil
	case actortypes.Version6:
		return v6.MinerHandler, nil
	case actortypes.Version7:
		return v7.MinerHandler, nil
	case actortypes.Version8:
		return v8.MinerHandler, nil
	case actortypes.Version9:
		return v9.MinerHandler, nil
	default:
		return nil, fmt.Errorf("unsupported miner actor version: %d", av)
	}
}
