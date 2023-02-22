package util

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	blocks "github.com/ipfs/go-libipfs/blocks"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/metrics"
)

type CacheConfig struct {
	BlockstoreCacheSize uint
	StatestoreCacheSize uint
}

var _ blockstore.Blockstore = (*CachingBlockstore)(nil)

type CachingBlockstore struct {
	cache  *lru.ARCCache
	blocks blockstore.Blockstore
	reads  int64 // updated atomically
	hits   int64 // updated atomically
	bytes  int64 // updated atomically
}

func NewCachingBlockstore(blocks blockstore.Blockstore, cacheSize int) (*CachingBlockstore, error) {
	metrics.RecordCount(context.TODO(), metrics.BlockStoreCacheLimit, cacheSize)
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("new arc: %w", err)
	}

	return &CachingBlockstore{
		cache:  cache,
		blocks: blocks,
	}, nil
}

func (cs *CachingBlockstore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	return cs.blocks.DeleteBlock(ctx, c)
}

func (cs *CachingBlockstore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	return cs.blocks.GetSize(ctx, c)
}

func (cs *CachingBlockstore) Put(ctx context.Context, blk blocks.Block) error {
	return cs.blocks.Put(ctx, blk)
}

func (cs *CachingBlockstore) PutMany(ctx context.Context, blks []blocks.Block) error {
	return cs.blocks.PutMany(ctx, blks)
}

func (cs *CachingBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return cs.blocks.AllKeysChan(ctx)
}

func (cs *CachingBlockstore) HashOnRead(enabled bool) {
	cs.blocks.HashOnRead(enabled)
}

func (cs *CachingBlockstore) DeleteMany(ctx context.Context, cids []cid.Cid) error {
	return cs.blocks.DeleteMany(ctx, cids)
}

func (cs *CachingBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	metrics.RecordCount(ctx, metrics.BlockStoreCacheSize, cs.cache.Len())
	metrics.RecordInc(ctx, metrics.BlockStoreCacheRead)
	reads := atomic.AddInt64(&cs.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cs.hits)
		by := atomic.LoadInt64(&cs.bytes)
		log.Debugw("CachingBlockstore stats", "reads", reads, "cache_len", cs.cache.Len(), "hit_rate", float64(hits)/float64(reads), "bytes_read", by)
	}

	v, hit := cs.cache.Get(c)
	if hit {
		metrics.RecordInc(ctx, metrics.BlockStoreCacheHits)
		atomic.AddInt64(&cs.hits, 1)
		return v.(blocks.Block), nil
	}

	blk, err := cs.blocks.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&cs.bytes, int64(len(blk.RawData())))
	cs.cache.Add(c, blk)
	return blk, err
}

func (cs *CachingBlockstore) View(ctx context.Context, c cid.Cid, callback func([]byte) error) error {
	metrics.RecordInc(ctx, metrics.BlockStoreCacheRead)
	reads := atomic.AddInt64(&cs.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cs.hits)
		by := atomic.LoadInt64(&cs.bytes)
		log.Debugw("CachingBlockstore stats", "reads", reads, "cache_len", cs.cache.Len(), "hit_rate", float64(hits)/float64(reads), "bytes_read", by)
	}
	v, hit := cs.cache.Get(c)
	if hit {
		metrics.RecordInc(ctx, metrics.BlockStoreCacheHits)
		atomic.AddInt64(&cs.hits, 1)
		return callback(v.(blocks.Block).RawData())
	}

	blk, err := cs.blocks.Get(ctx, c)
	if err != nil {
		return err
	}

	atomic.AddInt64(&cs.bytes, int64(len(blk.RawData())))
	cs.cache.Add(c, blk)
	return callback(blk.RawData())
}

func (cs *CachingBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	metrics.RecordInc(ctx, metrics.BlockStoreCacheRead)
	atomic.AddInt64(&cs.reads, 1)
	// Safe to query cache since blockstore never deletes
	if cs.cache.Contains(c) {
		return true, nil
	}

	return cs.blocks.Has(ctx, c)
}

var _ adt.Store = (*CachingStateStore)(nil)

type CachingStateStore struct {
	cache  *lru.ARCCache
	blocks blockstore.Blockstore
	store  adt.Store
	reads  int64 // updated atomically
	hits   int64 // updated atomically
}

func NewCachingStateStore(blocks blockstore.Blockstore, cacheSize int) (*CachingStateStore, error) {
	metrics.RecordCount(context.TODO(), metrics.StateStoreCacheLimit, cacheSize)
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("new arc: %w", err)
	}

	store := adt.WrapStore(context.Background(), cbor.NewCborStore(blocks))

	return &CachingStateStore{
		cache:  cache,
		store:  store,
		blocks: blocks,
	}, nil
}

func (cas *CachingStateStore) Context() context.Context {
	return context.Background()
}

func (cas *CachingStateStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	metrics.RecordCount(ctx, metrics.StateStoreCacheSize, cas.cache.Len())
	metrics.RecordInc(ctx, metrics.StateStoreCacheRead)
	reads := atomic.AddInt64(&cas.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cas.hits)
		log.Debugw("CachingStateStore stats", "reads", reads, "cache_len", cas.cache.Len(), "hit_rate", float64(hits)/float64(reads))
	}

	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return fmt.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	v, hit := cas.cache.Get(c)
	if hit {
		err := cas.tryAssign(v, out)
		if err == nil {
			metrics.RecordInc(ctx, metrics.StateStoreCacheHits)
			atomic.AddInt64(&cas.hits, 1)
			return nil
		}

		// log and fall through to get from store
		log.Debugw("CachingStateStore failed to read from cache", "error", err.Error())
	}

	blk, err := cas.blocks.Get(ctx, c)
	if err != nil {
		return err
	}

	if err := cu.UnmarshalCBOR(bytes.NewReader(blk.RawData())); err != nil {
		return cbor.NewSerializationError(err)
	}

	o := reflect.ValueOf(out).Elem()
	cas.cache.Add(c, o)
	return nil
}

func (cas *CachingStateStore) tryAssign(value interface{}, out interface{}) error {
	o := reflect.ValueOf(out).Elem()
	if !o.CanSet() {
		return fmt.Errorf("out parameter (type %s) cannot be set", o.Type().Name())
	}

	if !value.(reflect.Value).Type().AssignableTo(o.Type()) {
		return fmt.Errorf("out parameter (type %s) cannot be assigned cached value (type %s)", o.Type().Name(), value.(reflect.Value).Type().Name())
	}

	o.Set(value.(reflect.Value))
	return nil
}

func (cas *CachingStateStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cas.store.Put(ctx, v)
}
