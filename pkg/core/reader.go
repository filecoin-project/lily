package core

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
)

func StateReader(ctx context.Context, store adt.Store, c cid.Cid, fn interface{}) error {
	raw := new(typegen.Deferred)
	if err := store.Get(ctx, c, raw); err != nil {
		return err
	}
	return StateReadDeferred(ctx, raw, fn)
}

func StateReadDeferred(ctx context.Context, raw *typegen.Deferred, fn interface{}) error {
	fnArg := reflect.TypeOf(fn).In(0)
	if fnArg.Implements(reflect.TypeOf((*typegen.CBORUnmarshaler)(nil)).Elem()) {
		p := reflect.New(fnArg.Elem()).Interface().(typegen.CBORUnmarshaler)
		if err := p.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return err
		}
		results := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(p)})
		if results[0].IsNil() {
			return nil
		} else {
			return fmt.Errorf("error: %s", results[0])
		}
	}
	panic("here")
	return nil

}
