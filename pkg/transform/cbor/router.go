package cbor

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car/util"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	v1car "github.com/ipld/go-car"

	"github.com/filecoin-project/lily/pkg/extract/procesor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/init_"
	"github.com/filecoin-project/lily/pkg/transform/cbor/market"
	"github.com/filecoin-project/lily/pkg/transform/cbor/miner"
	"github.com/filecoin-project/lily/pkg/transform/cbor/power"
	"github.com/filecoin-project/lily/pkg/transform/cbor/verifreg"
)

var log = logging.Logger("lily/transform/cbor")

type ActorIPLDContainer struct {
	// TODO this needs to be versioned
	CurrentTipSet  *types.TipSet
	ExecutedTipSet *types.TipSet
	MinerActors    cid.Cid  // HAMT[Address]MinerStateChange
	VerifregActor  *cid.Cid // VerifregStateChange or empty
	ActorStates    cid.Cid  // HAMT[Address]ActorStateChange
	InitActor      cid.Cid  // HAMT[Address]AddressChanges.
	MarketActor    cid.Cid  // MarketStateChange or empty
	PowerActor     cid.Cid  // PowerStateChange or empty
}

func (a *ActorIPLDContainer) MarshalLogObject(enc zapcore.ObjectEncoder) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("current", a.CurrentTipSet.String()),
		attribute.String("executed", a.ExecutedTipSet.String()),
		attribute.String("miners", a.MinerActors.String()),
		attribute.String("verifreg", a.VerifregActor.String()),
		attribute.String("actors", a.ActorStates.String()),
		attribute.String("init", a.InitActor.String()),
		attribute.String("market", a.MarketActor.String()),
		attribute.String("power", a.PowerActor.String()),
	}
}

func ProcessState(ctx context.Context, changes *procesor.ActorStateChanges, w io.Writer) error {
	bs := blockstore.NewMemorySync()
	store := adt2.WrapBlockStore(ctx, bs)
	actorStates, err := ProcessActors(ctx, bs, changes)
	if err != nil {
		return err
	}

	actorStatesRoot, err := store.Put(ctx, actorStates)
	if err != nil {
		return err
	}
	log.Infow("Wrote Delta", "root", actorStatesRoot.String(), actorStates)
	if err := v1car.WriteHeader(&v1car.CarHeader{
		Roots:   []cid.Cid{actorStatesRoot},
		Version: 1,
	}, w); err != nil {
		return err
	}
	keyCh, err := bs.AllKeysChan(ctx)
	if err != nil {
		return err
	}
	for key := range keyCh {
		blk, err := bs.Get(ctx, key)
		if err != nil {
			return err
		}
		if err := util.LdWrite(w, blk.Cid().Bytes(), blk.RawData()); err != nil {
			return err
		}
	}
	return nil
}

func ProcessActors(ctx context.Context, bs blockstore.Blockstore, changes *procesor.ActorStateChanges) (*ActorIPLDContainer, error) {
	out := &ActorIPLDContainer{
		CurrentTipSet:  changes.Current,
		ExecutedTipSet: changes.Executed,
	}
	if changes.MinerActors != nil {
		minerRoot, err := miner.HandleChanges(ctx, bs, changes.MinerActors)
		if err != nil {
			return nil, err
		}
		out.MinerActors = minerRoot
	}

	if changes.VerifregActor != nil {
		verifregRoot, err := verifreg.HandleChanges(ctx, bs, changes.VerifregActor)
		if err != nil {
			return nil, err
		}
		out.VerifregActor = &verifregRoot
	}

	if changes.ActorStates != nil {
		actorsRoot, err := actor.HandleChanges(ctx, bs, changes.ActorStates)
		if err != nil {
			return nil, err
		}
		out.ActorStates = actorsRoot
	}

	if changes.InitActor != nil {
		initRoot, err := init_.HandleChanges(ctx, bs, changes.InitActor)
		if err != nil {
			return nil, err
		}
		out.InitActor = initRoot
	}

	if changes.MarketActor != nil {
		marketRoot, err := market.HandleChange(ctx, bs, changes.MarketActor)
		if err != nil {
			return nil, err
		}
		out.MarketActor = marketRoot
	}

	if changes.PowerActor != nil {
		powerRoot, err := power.HandleChange(ctx, bs, changes.PowerActor)
		if err != nil {
			return nil, err
		}
		out.PowerActor = powerRoot
	}
	return out, nil
}
