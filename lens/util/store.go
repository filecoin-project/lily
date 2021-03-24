package util

import (
	"context"
	"sync/atomic"

	"github.com/filecoin-project/lotus/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
)

func NewCachingStore(backing blockstore.Blockstore) *ProxyingBlockstore {
	cache := ds.NewMapDatastore()
	bs := blockstore.NewBlockstore(cache)

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
