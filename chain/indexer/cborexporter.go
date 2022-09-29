package indexer

/*
import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	carbs "github.com/ipld/go-car/v2/blockstore"
	typegen "github.com/whyrusleeping/cbor-gen"

	v2 "github.com/filecoin-project/lily/model/v2"
	messages2 "github.com/filecoin-project/lily/model/v2/messages"
)

const BitWidth = 8

type LilyModelStorage struct {
	bs    blockstore.Blockstore
	store adt.Store
}

func NewLilyModelStorage(ctx context.Context, bs blockstore.Blockstore) (*LilyModelStorage, error) {
	s := adt.WrapStore(ctx, cbor.NewCborStore(bs))
	return &LilyModelStorage{
		bs:    bs,
		store: s,
	}, nil
}

func ModelsAsCAR(ctx context.Context, f *os.File, stateroot cid.Cid, models []v2.LilyModel) error {
	// calculate the root cid of the model hamt
	// TODO this is hacky but we need to do it in order to calculate the model HAMT root to write the car file with.
	mbs := blockstore.NewMemory()
	memModelRoot, err := StoreModels(ctx, adt.WrapStore(ctx, cbor.NewCborStore(mbs)), stateroot, models)
	if err != nil {
		return err
	}

	// create a car file using the calculated model root. note this is a v2 car file.
	carrw, err := carbs.OpenReadWriteFile(f, []cid.Cid{memModelRoot})
	if err != nil {
		return err
	}

	// persist models to car file
	carModelRoot, err := StoreModels(ctx, adt.WrapStore(ctx, cbor.NewCborStore(carrw)), stateroot, models)
	if err != nil {
		return err
	}

	// sanity check.
	if !carModelRoot.Equals(memModelRoot) {
		return fmt.Errorf("calculated model root %s does not match car file model root %s", memModelRoot, carModelRoot)
	}

	return carrw.Finalize()
}

func StoreModels(ctx context.Context, store adt.Store, stateroot cid.Cid, models []v2.LilyModel) (cid.Cid, error) {
	modelRoot, err := PutModels(ctx, store, models)
	if err != nil {
		return cid.Undef, err
	}
	return PutModelsMap(ctx, store, modelRoot, stateroot)
}

func PutModelsMap(ctx context.Context, store adt.Store, modelroot, stateroot cid.Cid) (cid.Cid, error) {
	// create stateroot map
	// map[stateroot]map[modelType][]Model
	srm, err := adt.MakeEmptyMap(store, BitWidth)
	if err != nil {
		return cid.Undef, err
	}

	if err := srm.Put(abi.CidKey(stateroot), typegen.CborCid(modelroot)); err != nil {
		return cid.Undef, err
	}

	return srm.Root()
}

func PutModels(ctx context.Context, store adt.Store, models []v2.LilyModel) (cid.Cid, error) {
	// create model multimap
	// map[ModelType][]Model
	mmm, err := adt.MakeEmptyMultimap(store, BitWidth, BitWidth)
	if err != nil {
		return cid.Undef, err
	}
	// add all the models
	for _, model := range models {
		if err := mmm.Add(model, model); err != nil {
			return cid.Undef, err
		}
	}
	return mmm.Root()

}

func VMMessagesAtStateRoot(store adt.Store, root, stateroot cid.Cid) ([]messages2.VMMessage, error) {
	rootMap, err := adt.AsMap(store, root, BitWidth)
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
	mmap, err := adt.AsMultimap(store, cid.Cid(mmapRoot), BitWidth, BitWidth)
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


*/
