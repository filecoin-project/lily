package lotus

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/sentinel-visor/metrics"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

type CacheCtxStore struct {
	cache *lru.ARCCache
	ctx   context.Context
	api   v0api.FullNode
}

func NewCacheCtxStore(ctx context.Context, api v0api.FullNode, cache *lru.ARCCache) (*CacheCtxStore, error) {
	return &CacheCtxStore{
		cache: cache,
		ctx:   ctx,
		api:   api,
	}, nil
}

func (cs *CacheCtxStore) Context() context.Context {
	return cs.ctx
}

func (cs *CacheCtxStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	ctx, span := global.Tracer("").Start(ctx, "CacheCtxStore.Get")
	defer span.End()

	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return fmt.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	// hit :)
	v, hit := cs.cache.Get(c)
	if hit {
		return cu.UnmarshalCBOR(bytes.NewReader(v.([]byte)))
	}

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainReadObj"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()

	// miss :(
	raw, err := cs.api.ChainReadObj(ctx, c)
	if err != nil {
		return xerrors.Errorf("read obj: %w", err)
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(raw)); err != nil {
		return xerrors.Errorf("unmarshal obj: %w", err)
	}

	cs.cache.Add(c, raw)
	return nil
}

func (cs *CacheCtxStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cid.Undef, fmt.Errorf("put is not implemented on CacheCtxStore")
}
