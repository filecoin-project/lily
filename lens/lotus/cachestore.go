package lotus

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/api"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	"github.com/opentracing/opentracing-go"
	cbg "github.com/whyrusleeping/cbor-gen"
)

type CacheCtxStore struct {
	cache *lru.ARCCache
	ctx   context.Context
	api   api.FullNode
}

func NewCacheCtxStore(ctx context.Context, api api.FullNode, size int) *CacheCtxStore {
	ac, err := lru.NewARC(size)
	if err != nil {
		panic(err)
	}

	return &CacheCtxStore{
		cache: ac,
		ctx:   ctx,
		api:   api,
	}
}

func (cs *CacheCtxStore) Context() context.Context {
	return cs.ctx
}

func (cs *CacheCtxStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CacheCtxStore.Get")
	defer span.Finish()
	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return fmt.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	// hit :)
	v, hit := cs.cache.Get(c)
	if hit {
		return cu.UnmarshalCBOR(bytes.NewReader(v.([]byte)))
	}

	// miss :(
	raw, err := cs.api.ChainReadObj(ctx, c)
	if err != nil {
		return err
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		return err
	}

	cs.cache.Add(c, raw)
	return nil
}

func (cs *CacheCtxStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("put is not implemented on CacheCtxStore")
}
