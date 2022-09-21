package indexer

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	typegen "github.com/whyrusleeping/cbor-gen"

	v2 "github.com/filecoin-project/lily/model/v2"
	messages2 "github.com/filecoin-project/lily/model/v2/messages"
)

const BitWidth = 8

type LilyModelStorage struct {
	BS             blockstore.Blockstore
	Store          adt.Store
	ModelStateRoot cid.Cid
}

func NewLilyModelStorage(ctx context.Context, bs blockstore.Blockstore) (*LilyModelStorage, error) {
	s := adt.WrapStore(ctx, cbor.NewCborStore(bs))
	modelRoot, err := adt.StoreEmptyMap(s, BitWidth)
	if err != nil {
		return nil, err
	}
	return &LilyModelStorage{
		BS:             bs,
		Store:          s,
		ModelStateRoot: modelRoot,
	}, nil

}

func (lms *LilyModelStorage) PersistModels(ctx context.Context, stateroot cid.Cid, models []v2.LilyModel) error {
	mmapRoot, err := adt.StoreEmptyMultimap(lms.Store, BitWidth, BitWidth)
	if err != nil {
		return err
	}
	mmap, err := adt.AsMultimap(lms.Store, mmapRoot, BitWidth, BitWidth)
	if err != nil {
		return err
	}
	for _, model := range models {
		if err := mmap.Add(model, model); err != nil {
			return err
		}
	}
	mmapRoot, err = mmap.Root()
	if err != nil {
		return err
	}

	rootMap, err := adt.AsMap(lms.Store, lms.ModelStateRoot, BitWidth)
	if err != nil {
		return err
	}

	if err := rootMap.Put(abi.CidKey(stateroot), typegen.CborCid(mmapRoot)); err != nil {
		return err
	}
	newModelRoot, err := rootMap.Root()
	if err != nil {
		return err
	}
	lms.ModelStateRoot = newModelRoot

	log.Infow("EXPORTED MODELS", "root", lms.ModelStateRoot.String())

	msgs, err := lms.VMMessagesAtStateRoot(stateroot)
	if err != nil {
		return err
	}
	_ = msgs
	return nil
}

func (lms *LilyModelStorage) VMMessagesAtStateRoot(stateroot cid.Cid) ([]messages2.VMMessage, error) {
	rootMap, err := adt.AsMap(lms.Store, lms.ModelStateRoot, BitWidth)
	if err != nil {
		return nil, err
	}
	var mmapRoot typegen.CborCid
	found, err := rootMap.Get(abi.CidKey(stateroot), &mmapRoot)
	if err != nil {
		return nil, err
	}
	if !found {
		panic("here")
	}
	mmap, err := adt.AsMultimap(lms.Store, cid.Cid(mmapRoot), BitWidth, BitWidth)
	if err != nil {
		return nil, err
	}
	var out []messages2.VMMessage
	var vmmsg messages2.VMMessage
	if err := mmap.ForEach(&messages2.VMMessage{}, &vmmsg, func(i int64) error {
		out = append(out, vmmsg)
		return nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}
