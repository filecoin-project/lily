package cbor

import (
	"context"
	"io"

	adtStore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"

	"github.com/filecoin-project/lily/pkg/extract/processor"
	cboractors "github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	messages2 "github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

var log = logging.Logger("lily/transform/cbor")

func Process(ctx context.Context, messages *processor.MessageStateChanges, actors *processor.ActorStateChanges, w io.Writer) error {
	bs := blockstore.NewMemorySync()
	store := adtStore.WrapBlockStore(ctx, bs)

	messageStateContainer, err := messages2.ProcessMessages(ctx, store, messages)
	if err != nil {
		return err
	}
	messageStatesRoot, err := store.Put(ctx, messageStateContainer)
	if err != nil {
		return err
	}

	// TODO pass the adtStore not the blockstore.
	actorStateContainer, err := cboractors.ProcessActorsStates(ctx, bs, actors)
	if err != nil {
		return err
	}
	actorStatesRoot, err := store.Put(ctx, actorStateContainer)
	if err != nil {
		return err
	}

	if err := v1car.WriteHeader(&v1car.CarHeader{
		Roots:   []cid.Cid{actorStatesRoot, messageStatesRoot},
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
