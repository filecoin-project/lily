package actors

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/raw"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward"
)

var log = logging.Logger("/lily/transform/timescale")

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

	actorIPLDContainer := new(cbor.ActorIPLDContainer)
	if err := adtStore.Get(ctx, header.Roots[0], actorIPLDContainer); err != nil {
		return err
	}

	log.Infow("Open Delta", "root", header.Roots[0], zap.Inline(actorIPLDContainer))

	current := actorIPLDContainer.CurrentTipSet
	executed := actorIPLDContainer.ExecutedTipSet
	av, err := core.ActorVersionForTipSet(ctx, current, nvg)
	if err != nil {
		return err
	}

	var models model.PersistableList
	actorModels, err := ProcessActorStates(ctx, bs, current, executed, av, actorIPLDContainer.ActorStates)
	if err != nil {
		return err
	}
	models = append(models, actorModels)

	minerModels, err := ProcessMiners(ctx, bs, current, executed, av, actorIPLDContainer.MinerActors)
	if err != nil {
		return err
	}
	models = append(models, minerModels)

	return strg.PersistBatch(ctx, models...)
}

func ProcessMiners(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	minerHandler, err := miner.MakeMinerProcessor(av)
	if err != nil {
		return nil, err
	}
	return minerHandler(ctx, bs, current, executed, root)
}

func ProcessActorStates(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, av actorstypes.Version, actorMapRoot cid.Cid) (model.Persistable, error) {
	var out = model.PersistableList{}
	adtStore := store.WrapBlockStore(ctx, bs)
	actorMap, err := adt.AsMap(adtStore, actorMapRoot, 5)
	if err != nil {
		return nil, err
	}
	actorState := new(actordiff.ActorChange)
	if err := actorMap.ForEach(actorState, func(key string) error {
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}

		m, err := raw.RawActorHandler(ctx, current, executed, addr, actorState)
		if err != nil {
			return err
		}
		if m != nil {
			out = append(out, m)
		}

		if core.RewardCodes.Has(actorState.Actor.Code) {
			m, err := reward.HandleReward(ctx, current, executed, addr, actorState, av)
			if err != nil {
				return err
			}
			out = append(out, m)
		}

		if core.MinerCodes.Has(actorState.Actor.Code) {
			m, err := miner.HandleMiner(ctx, current, executed, addr, actorState, av)
			if err != nil {
				return err
			}
			out = append(out, m)
		}
		return nil

	}); err != nil {
		return nil, err
	}
	return out, nil
}
