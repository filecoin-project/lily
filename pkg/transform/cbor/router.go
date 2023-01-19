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
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/pkg/extract/processor"
	cboractors "github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	messages2 "github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

var log = logging.Logger("lily/transform/cbor")

type RootStateIPLD struct {
	StateVersion uint64 `cborgen:"stateversion"`

	NetworkName    string `cborgen:"networkname"`
	NetworkVersion uint64 `cborgen:"networkversion"`

	State cid.Cid `cborgen:"state"` // StateExtractionIPLD
}

func (r *RootStateIPLD) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("state_version", int64(r.StateVersion)),
		attribute.String("state_root", r.State.String()),
		attribute.String("network_name", r.NetworkName),
		attribute.Int64("network_version", int64(r.NetworkVersion)),
	}
}

func (r *RootStateIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range r.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

type StateExtractionIPLD struct {
	Current types.TipSet `cborgen:"current"`
	Parent  types.TipSet `cborgen:"parent"`

	BaseFee abi.TokenAmount `cborgen:"basefee"`

	FullBlocks       cid.Cid `cborgen:"fullblocks"`
	ImplicitMessages cid.Cid `cborgen:"implicitmessages"`
	Actors           cid.Cid `cborgen:"actors"`
}

func (s *StateExtractionIPLD) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("current_tipset", s.Current.Key().String()),
		attribute.String("parent_tipset", s.Parent.Key().String()),
		attribute.String("base_fee", s.BaseFee.String()),
		attribute.String("full_block_root", s.FullBlocks.String()),
		attribute.String("implicit_message_root", s.ImplicitMessages.String()),
		attribute.String("actors_root", s.Actors.String()),
	}
}

func (s *StateExtractionIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range s.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
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

	implicitMsgsAMT, err := messages2.MakeImplicitMessagesHAMT(ctx, store, messages.ImplicitMessages)
	if err != nil {
		return cid.Undef, err
	}

	fullBlkHAMT, err := messages2.MakeFullBlockHAMT(ctx, store, messages.FullBlocks)
	if err != nil {
		return cid.Undef, err
	}

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
