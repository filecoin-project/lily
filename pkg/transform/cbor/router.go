package cbor

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car/util"

	v1car "github.com/ipld/go-car"

	"github.com/filecoin-project/lily/pkg/extract/procesor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/actor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/init_"
	"github.com/filecoin-project/lily/pkg/transform/cbor/miner"
	"github.com/filecoin-project/lily/pkg/transform/cbor/verifreg"
)

type ActorIPLDContainer struct {
	// TODO this needs to be versioned
	CurrentTipSet  *types.TipSet
	ExecutedTipSet *types.TipSet
	MinerActors    cid.Cid  // HAMT[Address]MinerStateChange
	VerifregActor  *cid.Cid // VerifregStateChange or empty
	ActorStates    cid.Cid  // HAMT[Address]ActorStateChange
	InitActor      cid.Cid  // HAMT[Address]AddressChanges.
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
	minerRoot, err := miner.HandleChanges(ctx, bs, changes.MinerActors)
	if err != nil {
		return nil, err
	}
	out.MinerActors = minerRoot

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
	return out, nil
}
