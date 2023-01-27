package timescale

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
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	"github.com/filecoin-project/lily/pkg/transform/cbor/messages"
	init_ "github.com/filecoin-project/lily/pkg/transform/timescale/actors/init"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/market"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/power"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/raw"
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

func HandleActorStateChanges(ctx context.Context, s store.Store, current, parent *types.TipSet, av actorstypes.Version, root cid.Cid) (model.Persistable, error) {
	actorIPLDContainer := new(actors.ActorStateChangesIPLD)
	if err := s.Get(ctx, root, actorIPLDContainer); err != nil {
		return nil, err
	}
	log.Infow("open actor state changes", zap.Inline(actorIPLDContainer))

	out := model.PersistableList{}
	marketModels, err := market.TransformMarketState(ctx, s, av, current, parent, actorIPLDContainer.MarketActor)
	if err != nil {
		return nil, err
	}
	out = append(out, marketModels)

	minerModels, err := miner.TransformMinerState(ctx, s, av, current, parent, actorIPLDContainer.MinerActors)
	if err != nil {
		return nil, err
	}
	out = append(out, minerModels)

	powerModels, err := power.TransformPowerState(ctx, s, av, current, parent, actorIPLDContainer.PowerActor)
	if err != nil {
		return nil, err
	}
	out = append(out, powerModels)

	initModels, err := init_.TransformInitState(ctx, s, current, parent, actorIPLDContainer.InitActor)
	if err != nil {
		return nil, err
	}
	out = append(out, initModels)

	verifregModels, err := verifreg.TransformVerifregState(ctx, s, av, current, parent, actorIPLDContainer.VerifregActor)
	if err != nil {
		return nil, err
	}
	out = append(out, verifregModels)

	rawModels, err := raw.TransformActorStates(ctx, s, current, parent, actorIPLDContainer.RawActors)
	if err != nil {
		return nil, err
	}
	out = append(out, rawModels)

	return out, nil
}

func HandleFullBlocks(ctx context.Context, s store.Store, current, parent *types.TipSet, root cid.Cid) (model.PersistableList, error) {
	out := model.PersistableList{}
	fullBlockMap, err := messages.DecodeFullBlockHAMT(ctx, s, root)
	if err != nil {
		return nil, err
	}
	out = append(out, fullblock.ExtractBlockHeaders(ctx, fullBlockMap))
	out = append(out, fullblock.ExtractBlockParents(ctx, fullBlockMap))
	out = append(out, fullblock.ExtractBlockMessages(ctx, fullBlockMap))
	out = append(out, fullblock.ExtractMessages(ctx, fullBlockMap))
	out = append(out, fullblock.ExtractVmMessages(ctx, fullBlockMap))

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
