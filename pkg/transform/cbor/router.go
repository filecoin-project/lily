package cbor

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-state-types/abi"
	adtStore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"

	"github.com/filecoin-project/lily/pkg/extract/processor"
	cboractors "github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	messages2 "github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

var log = logging.Logger("lily/transform/cbor")

type RootStateIPLD struct {
	StateVersion uint64

	State cid.Cid
}

type StateExtractionIPLD struct {
	Current types.TipSet
	Parent  types.TipSet

	BaseFee abi.TokenAmount

	FullBlocks       cid.Cid
	ImplicitMessages cid.Cid
	Actors           cid.Cid
}

func WriteCar(ctx context.Context, root cid.Cid, carVersion uint64, bs blockstore.Blockstore, w io.Writer) error {
	if err := v1car.WriteHeader(&v1car.CarHeader{
		Roots:   []cid.Cid{root},
		Version: carVersion,
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

func PersistToStore(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, messages *processor.MessageStateChanges, actors *processor.ActorStateChanges) (cid.Cid, error) {
	store := adtStore.WrapBlockStore(ctx, bs)

	// sanity check
	if !messages.Current.Equals(actors.Current) {
		return cid.Undef, fmt.Errorf("actor and message current tipset does not match")
	}
	if !messages.Executed.Equals(actors.Executed) {
		return cid.Undef, fmt.Errorf("actor and message executed tipset does not match")
	}

	implicitMsgsAMT, err := messages2.MakeImplicitMessagesAMT(ctx, store, messages.ImplicitMessages)
	if err != nil {
		return cid.Undef, err
	}

	fullBlkHAMT, err := messages2.MakeFullBlockHAMT(ctx, store, messages.FullBlocks)
	if err != nil {
		return cid.Undef, err
	}

	// TODO pass the adtStore not the blockstore.
	actorStateContainer, err := cboractors.ProcessActorsStates(ctx, store, actors)
	if err != nil {
		return cid.Undef, err
	}

	actorStatesRoot, err := store.Put(ctx, actorStateContainer)
	if err != nil {
		return cid.Undef, err
	}

	extractedState := &StateExtractionIPLD{
		Current:          *current,
		Parent:           *executed,
		BaseFee:          messages.BaseFee,
		FullBlocks:       fullBlkHAMT,
		ImplicitMessages: implicitMsgsAMT,
		Actors:           actorStatesRoot,
	}

	extractedStateRoot, err := store.Put(ctx, extractedState)
	if err != nil {
		return cid.Undef, err
	}

	rootState := &RootStateIPLD{
		StateVersion: 0,
		State:        extractedStateRoot,
	}

	root, err := store.Put(ctx, rootState)
	if err != nil {
		return cid.Undef, err
	}
	return root, nil
}
