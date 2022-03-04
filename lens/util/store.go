package util

import (
	"bytes"
	"context"
	"reflect"
	"sync/atomic"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

type CacheConfig struct {
	BlockstoreCacheSize uint
	StatestoreCacheSize uint
}

func NewCachingStore(backing blockstore.Blockstore) *ProxyingBlockstore {
	bs := blockstore.NewMemorySync()

	return &ProxyingBlockstore{
		cache: bs,
		store: backing,
	}
}

type ProxyingBlockstore struct {
	cache blockstore.Blockstore
	store blockstore.Blockstore
	gets  int64 // updated atomically
}

func (pb *ProxyingBlockstore) View(key cid.Cid, callback func([]byte) error) error {
	if err := pb.cache.View(key, callback); err == nil {
		return nil
	}
	return pb.store.View(key, callback)
}

func (pb *ProxyingBlockstore) DeleteMany(keys []cid.Cid) error {
	return pb.cache.DeleteMany(keys)
}

func (pb *ProxyingBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	atomic.AddInt64(&pb.gets, 1)
	if block, err := pb.cache.Get(c); err == nil {
		return block, err
	}

	return pb.store.Get(c)
}

func (pb *ProxyingBlockstore) Has(c cid.Cid) (bool, error) {
	if h, err := pb.cache.Has(c); err == nil && h {
		return true, nil
	}

	return pb.store.Has(c)
}

func (pb *ProxyingBlockstore) DeleteBlock(c cid.Cid) error {
	return pb.cache.DeleteBlock(c)
}

func (pb *ProxyingBlockstore) GetSize(c cid.Cid) (int, error) {
	if s, err := pb.cache.GetSize(c); err == nil {
		return s, nil
	}
	return pb.store.GetSize(c)
}

func (pb *ProxyingBlockstore) Put(b blocks.Block) error {
	return pb.cache.Put(b)
}

func (pb *ProxyingBlockstore) PutMany(bs []blocks.Block) error {
	for _, b := range bs {
		if err := pb.Put(b); err != nil {
			return err
		}
	}
	return nil
}

func (pb *ProxyingBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	outChan := make(chan cid.Cid, 10)

	cctx, cncl := context.WithCancel(ctx)
	akc, err := pb.cache.AllKeysChan(cctx)
	if err != nil {
		cncl()
		return nil, err
	}
	akc2, err2 := pb.store.AllKeysChan(cctx)
	if err2 != nil {
		cncl()
		return nil, err2
	}
	go func() {
		defer cncl()
		defer close(outChan)
		for c := range akc {
			outChan <- c
		}
		for c := range akc2 {
			outChan <- c
		}
	}()

	return outChan, nil
}

func (pb *ProxyingBlockstore) HashOnRead(enabled bool) {
}

func (pb *ProxyingBlockstore) GetCount() int64 {
	c := atomic.LoadInt64(&pb.gets)
	return c
}

func (pb *ProxyingBlockstore) ResetMetrics() {
	atomic.StoreInt64(&pb.gets, 0)
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
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, xerrors.Errorf("new arc: %w", err)
	}

	return &CachingBlockstore{
		cache:  cache,
		blocks: blocks,
	}, nil
}

func (cs *CachingBlockstore) DeleteBlock(c cid.Cid) error {
	return cs.blocks.DeleteBlock(c)
}

func (cs *CachingBlockstore) GetSize(c cid.Cid) (int, error) {
	return cs.blocks.GetSize(c)
}

func (cs *CachingBlockstore) Put(blk blocks.Block) error {
	return cs.blocks.Put(blk)
}

func (cs *CachingBlockstore) PutMany(blks []blocks.Block) error {
	return cs.blocks.PutMany(blks)
}

func (cs *CachingBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return cs.blocks.AllKeysChan(ctx)
}

func (cs *CachingBlockstore) HashOnRead(enabled bool) {
	cs.blocks.HashOnRead(enabled)
}

func (cs *CachingBlockstore) DeleteMany(cids []cid.Cid) error {
	return cs.blocks.DeleteMany(cids)
}

func (cs *CachingBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	reads := atomic.AddInt64(&cs.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cs.hits)
		by := atomic.LoadInt64(&cs.bytes)
		log.Debugw("CachingBlockstore stats", "reads", reads, "cache_len", cs.cache.Len(), "hit_rate", float64(hits)/float64(reads), "bytes_read", by)
	}

	v, hit := cs.cache.Get(c)
	if hit {
		atomic.AddInt64(&cs.hits, 1)
		return v.(blocks.Block), nil
	}

	blk, err := cs.blocks.Get(c)
	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&cs.bytes, int64(len(blk.RawData())))
	cs.cache.Add(c, blk)
	return blk, err
}

func (cs *CachingBlockstore) View(c cid.Cid, callback func([]byte) error) error {
	reads := atomic.AddInt64(&cs.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cs.hits)
		by := atomic.LoadInt64(&cs.bytes)
		log.Debugw("CachingBlockstore stats", "reads", reads, "cache_len", cs.cache.Len(), "hit_rate", float64(hits)/float64(reads), "bytes_read", by)
	}
	v, hit := cs.cache.Get(c)
	if hit {
		atomic.AddInt64(&cs.hits, 1)
		return callback(v.(blocks.Block).RawData())
	}

	blk, err := cs.blocks.Get(c)
	if err != nil {
		return err
	}

	atomic.AddInt64(&cs.bytes, int64(len(blk.RawData())))
	cs.cache.Add(c, blk)
	return callback(blk.RawData())
}

func (cs *CachingBlockstore) Has(c cid.Cid) (bool, error) {
	atomic.AddInt64(&cs.reads, 1)
	// Safe to query cache since blockstore never deletes
	if cs.cache.Contains(c) {
		return true, nil
	}

	return cs.blocks.Has(c)
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
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		return nil, xerrors.Errorf("new arc: %w", err)
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
	reads := atomic.AddInt64(&cas.reads, 1)
	if reads%1000000 == 0 {
		hits := atomic.LoadInt64(&cas.hits)
		log.Debugw("CachingStateStore stats", "reads", reads, "cache_len", cas.cache.Len(), "hit_rate", float64(hits)/float64(reads))
	}

	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return xerrors.Errorf("out parameter does not implement CBORUnmarshaler")
	}

	v, hit := cas.cache.Get(c)
	if hit {
		err := cas.tryAssign(v, out)
		if err == nil {
			atomic.AddInt64(&cas.hits, 1)
			return nil
		}

		// log and fall through to get from store
		log.Debugw("CachingStateStore failed to read from cache", "error", err.Error())
	}

	blk, err := cas.blocks.Get(c)
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
		return xerrors.Errorf("out parameter (type %s) cannot be set", o.Type().Name())
	}

	if !value.(reflect.Value).Type().AssignableTo(o.Type()) {
		return xerrors.Errorf("out parameter (type %s) cannot be assigned cached value (type %s)", o.Type().Name(), value.(reflect.Value).Type().Name())
	}

	o.Set(value.(reflect.Value))
	return nil
}

func (cas *CachingStateStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return cas.store.Put(ctx, v)
}
