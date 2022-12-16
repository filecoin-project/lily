package actors

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	v1car "github.com/ipld/go-car"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v0"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v2"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v9"
)

type NetworkVersionGetter = func(ctx context.Context, epoch abi.ChainEpoch) network.Version

func Process(ctx context.Context, r io.Reader, strg model.Storage, nvg NetworkVersionGetter) error {
	bs := blockstore.NewMemorySync()
	header, err := v1car.LoadCar(ctx, bs, r)
	if err != nil {
		return err
	}
	if len(header.Roots) != 1 {
		return fmt.Errorf("invalid header expected 1 root got %d", len(header.Roots))
	}

	adtStore := store.WrapBlockStore(ctx, bs)

	var actorIPLDContainer cbor.ActorIPLDContainer
	if err := adtStore.Get(ctx, header.Roots[0], &actorIPLDContainer); err != nil {
		return err
	}

	current := actorIPLDContainer.CurrentTipSet
	executed := actorIPLDContainer.ExecutedTipSet
	av, err := core.ActorVersionForTipSet(ctx, current, nvg)
	if err != nil {
		return err
	}
	mapHandler := MakeMinerProcessor(av)
	minerModels, err := mapHandler(ctx, bs, current, executed, actorIPLDContainer.MinerActors)
	if err != nil {
		return err
	}

	return strg.PersistBatch(ctx, minerModels...)
}

type MinerHandler = func(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, minerMapRoot cid.Cid) (model.PersistableList, error)

func MakeMinerProcessor(av actorstypes.Version) MinerHandler {
	switch av {
	case actorstypes.Version0:
		return v0.MinerHandler
	case actorstypes.Version2:
		return v2.MinerHandler
	case actorstypes.Version9:
		return v9.MinerHandler
	}
	panic("developer error")
}
