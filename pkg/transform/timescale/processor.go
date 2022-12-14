package timescale

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	v1car "github.com/ipld/go-car"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/transform/cbor"
	cborminer "github.com/filecoin-project/lily/pkg/transform/cbor/miner"
)

func Process(ctx context.Context, r io.Reader) error {
	bs := blockstore.NewMemorySync()
	header, err := v1car.LoadCar(ctx, bs, r)
	if err != nil {
		return err
	}
	if len(header.Roots) != 1 {
		return fmt.Errorf("invalid header expected 1 root got %d", len(header.Roots))
	}
	store := store.WrapBlockStore(ctx, bs)
	var actorIPLDContainer cbor.ActorIPLDContainer
	if err := store.Get(ctx, header.Roots[0], &actorIPLDContainer); err != nil {
		return err
	}
	fmt.Println("miner hamt", actorIPLDContainer.MinerActors.String())
	minerMap, err := adt.AsMap(store, actorIPLDContainer.MinerActors, 5)
	if err != nil {
		return err
	}
	var minerState cborminer.StateChange
	if err := minerMap.ForEach(&minerState, func(key string) error {
		minerAddr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		fmt.Println(minerAddr.String())
		if minerState.Info != nil {
			HandleMinerInfo(ctx, store, *minerState.Info)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func HandleMinerInfo(ctx context.Context, store adt.Store, info cid.Cid) (interface{}, error) {
	var minerInfoChange cborminer.Info
	if err := store.Get(ctx, info, &minerInfoChange); err != nil {
		return nil, err
	}
	StateReader(ctx, store, minerInfoChange.Info, func(in *miner9.MinerInfo) error {
		fmt.Printf("%+v", in)
		return nil
	})
	return nil, nil
}

func StateReader(ctx context.Context, store adt.Store, c cid.Cid, fn interface{}) error {
	var tmp typegen.Deferred
	if err := store.Get(ctx, c, &tmp); err != nil {
		return err
	}
	fnArg := reflect.TypeOf(fn).In(0)
	fmt.Println(fnArg.String())
	if fnArg.Implements(reflect.TypeOf((*typegen.CBORUnmarshaler)(nil)).Elem()) {
		p := reflect.New(fnArg.Elem()).Interface().(typegen.CBORUnmarshaler)
		if err := p.UnmarshalCBOR(bytes.NewReader(tmp.Raw)); err != nil {
			return err
		}
		results := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(p)})
		fmt.Println(results[0].Interface())
	}
	return nil
}
