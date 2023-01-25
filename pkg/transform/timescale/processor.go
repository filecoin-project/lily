package timescale

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
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	"github.com/filecoin-project/lily/pkg/transform/cbor/messages"
	init_ "github.com/filecoin-project/lily/pkg/transform/timescale/actors/init"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/market"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/raw"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/reward"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg"
	"github.com/filecoin-project/lily/pkg/transform/timescale/fullblock"
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

	rootIPLDContainer := new(cbor.RootStateIPLD)
	if err := adtStore.Get(ctx, header.Roots[0], rootIPLDContainer); err != nil {
		return err
	}
	log.Infow("open root", "car_root", header.Roots[0], zap.Inline(rootIPLDContainer))

	stateExtractionIPLDContainer := new(cbor.StateExtractionIPLD)
	if err := adtStore.Get(ctx, rootIPLDContainer.State, stateExtractionIPLDContainer); err != nil {
		return err
	}
	log.Infow("open state extraction", zap.Inline(stateExtractionIPLDContainer))
	current := &stateExtractionIPLDContainer.Current
	parent := &stateExtractionIPLDContainer.Parent

	av, err := core.ActorVersionForTipSet(ctx, current, nvg)
	if err != nil {
		return err
	}

	toStorage := model.PersistableList{}
	fbModels, err := HandleFullBlocks(ctx, adtStore, current, parent, stateExtractionIPLDContainer.FullBlocks)
	if err != nil {
		return err
	}

	toStorage = append(toStorage, fbModels)

	_, err = HandleImplicitMessages(ctx, adtStore, current, parent, stateExtractionIPLDContainer.ImplicitMessages)
	if err != nil {
		return err
	}

	actorModels, err := HandleActorStateChanges(ctx, adtStore, current, parent, av, stateExtractionIPLDContainer.Actors)
	if err != nil {
		return err
	}

	toStorage = append(toStorage, actorModels)

	return strg.PersistBatch(ctx, toStorage)
}

func ProcessMiners(ctx context.Context, s store.Store, current, executed *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	minerHandler, err := miner.MakeMinerProcessor(av)
	if err != nil {
		return nil, err
	}
	return minerHandler(ctx, s, current, executed, root)
}

func ProcessInitAddresses(ctx context.Context, s store.Store, current, executed *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	return init_.InitHandler(ctx, s, current, executed, root)
}

func ProcessMarketActor(ctx context.Context, s store.Store, current, executed *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	marketHandler, err := market.MakeMarketProcessor(av)
	if err != nil {
		return nil, err
	}
	return marketHandler(ctx, s, current, executed, root)
}

func ProcessVerifregActor(ctx context.Context, s store.Store, current, executed *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	verifregHandler, err := verifreg.MakeVerifregProcessor(av)
	if err != nil {
		return nil, err
	}
	return verifregHandler(ctx, s, current, executed, root)
}

func HandleActorStateChanges(ctx context.Context, s store.Store, current, parent *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	actorIPLDContainer := new(actors.ActorStateChangesIPLD)
	if err := s.Get(ctx, root, actorIPLDContainer); err != nil {
		return nil, err
	}
	log.Infow("open actor state changes", zap.Inline(actorIPLDContainer))
	out := model.PersistableList{}
	if actorIPLDContainer.MarketActor != nil {
		marketModels, err := ProcessMarketActor(ctx, s, current, parent, av, *actorIPLDContainer.MarketActor)
		if err != nil {
			return nil, err
		}
		out = append(out, marketModels)
	}
	if actorIPLDContainer.MinerActors != nil {
		minerModels, err := ProcessMiners(ctx, s, current, parent, av, *actorIPLDContainer.MinerActors)
		if err != nil {
			return nil, err
		}
		out = append(out, minerModels)
	}
	if actorIPLDContainer.InitActor != nil {
		initModels, err := ProcessInitAddresses(ctx, s, current, parent, av, *actorIPLDContainer.InitActor)
		if err != nil {
			return nil, err
		}
		out = append(out, initModels)
	}
	if actorIPLDContainer.RawActors != nil {
		rawModels, err := ProcessActorStates(ctx, s, current, parent, av, *actorIPLDContainer.RawActors)
		if err != nil {
			return nil, err
		}
		out = append(out, rawModels)
	}
	if actorIPLDContainer.VerifregActor != nil {
		verifregModels, err := ProcessVerifregActor(ctx, s, current, parent, av, *actorIPLDContainer.VerifregActor)
		if err != nil {
			return nil, err
		}
		out = append(out, verifregModels)
	}

	return out, nil
}

func HandleFullBlocks(ctx context.Context, s store.Store, current, parent *types.TipSet, root cid.Cid) (model.PersistableList, error) {
	out := model.PersistableList{}
	fullBlockMap, err := messages.DecodeFullBlockHAMT(ctx, s, root)
	if err != nil {
		return nil, err
	}
	bh, err := fullblock.ExtractBlockHeaders(ctx, fullBlockMap)
	if err != nil {
		return nil, err
	}
	out = append(out, bh)
	bp, err := fullblock.ExtractBlockParents(ctx, fullBlockMap)
	if err != nil {
		return nil, err
	}
	out = append(out, bp)
	msgs, err := fullblock.ExtractMessages(ctx, fullBlockMap)
	if err != nil {
		return nil, err
	}
	out = append(out, msgs)
	vm, err := fullblock.ExtractVmMessages(ctx, fullBlockMap)
	if err != nil {
		return nil, err
	}
	out = append(out, vm)

	return out, nil
}

func HandleImplicitMessages(ctx context.Context, s store.Store, current, parent *types.TipSet, root cid.Cid) (interface{}, error) {
	implicitMessages, err := messages.DecodeImplicitMessagesHAMT(ctx, s, root)
	if err != nil {
		return nil, err
	}
	_ = implicitMessages
	return nil, nil
}

func ProcessActorStates(ctx context.Context, s store.Store, current, executed *types.TipSet, av actorstypes.Version, actorMapRoot cid.Cid) (model.Persistable, error) {
	var out = model.PersistableList{}
	actorMap, err := adt.AsMap(s, actorMapRoot, 5)
	if err != nil {
		return nil, err
	}
	actorState := new(rawdiff.ActorChange)
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
