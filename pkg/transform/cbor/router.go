package cbor

import (
	"context"
	"io"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	adt2 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car/util"

	"github.com/filecoin-project/lily/pkg/extract/procesor"
	"github.com/filecoin-project/lily/pkg/transform/cbor/miner"

	v1car "github.com/ipld/go-car"
)

type ActorIPLDContainer struct {
	CurrentTipSet  *types.TipSet
	ExecutedTipSet *types.TipSet
	MinerActors    cid.Cid // HAMT[Address]MinerStateChange
}

func ProcessState(ctx context.Context, changes *procesor.ActorStateChanges, w io.Writer) error {
	bs := blockstore.NewMemorySync()
	store := adt2.WrapBlockStore(ctx, bs)
	actorStates, err := ProcessActors(ctx, store, changes)
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

func ProcessActors(ctx context.Context, store adt.Store, changes *procesor.ActorStateChanges) (*ActorIPLDContainer, error) {
	m, err := adt.MakeEmptyMap(store, 5 /*TODO*/)
	if err != nil {
		return nil, err
	}
	if err := miner.HandleChanges(ctx, store, m, changes.MinerActors); err != nil {
		return nil, err
	}
	minerActorsRoot, err := m.Root()
	if err != nil {
		return nil, err
	}
	return &ActorIPLDContainer{
		CurrentTipSet:  changes.Current,
		ExecutedTipSet: changes.Executed,
		MinerActors:    minerActorsRoot,
	}, nil
}
