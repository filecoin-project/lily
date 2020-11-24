package util

import (
	"context"

	"github.com/filecoin-project/lotus/lib/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
)

func NewCachingStore(backing blockstore.Blockstore) blockstore.Blockstore {
	cache := ds.NewMapDatastore()
	bs := blockstore.NewBlockstore(cache)

	return &proxyingBlockstore{
		cache: bs,
		store: backing,
	}
}

type proxyingBlockstore struct {
	cache blockstore.Blockstore
	store blockstore.Blockstore
}

func (pb *proxyingBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	if block, err := pb.cache.Get(c); err == nil {
		return block, err
	}

	return pb.store.Get(c)
}

func (pb *proxyingBlockstore) Has(c cid.Cid) (bool, error) {
	if h, err := pb.cache.Has(c); err == nil && h {
		return true, nil
	}

	return pb.store.Has(c)
}

func (pb *proxyingBlockstore) DeleteBlock(c cid.Cid) error {
	return pb.cache.DeleteBlock(c)
}

func (pb *proxyingBlockstore) GetSize(c cid.Cid) (int, error) {
	if s, err := pb.cache.GetSize(c); err == nil {
		return s, nil
	}
	return pb.store.GetSize(c)
}

func (pb *proxyingBlockstore) Put(b blocks.Block) error {
	return pb.cache.Put(b)
}

func (pb *proxyingBlockstore) PutMany(bs []blocks.Block) error {
	for _, b := range bs {
		if err := pb.Put(b); err != nil {
			return err
		}
	}
	return nil
}

func (pb *proxyingBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	outChan := make(chan cid.Cid, 10)

	cctx, cncl := context.WithCancel(ctx)
	akc, err := pb.cache.AllKeysChan(cctx)
	if err != nil {
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

func (pb *proxyingBlockstore) HashOnRead(enabled bool) {
	return
}
