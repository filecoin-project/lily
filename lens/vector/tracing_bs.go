package vector

import (
	"github.com/filecoin-project/lotus/lib/blockstore"
	"github.com/filecoin-project/lotus/node/impl"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"sync"
)

func NewTracingBlockstore(bs blockstore.Blockstore) *TracingBlockstore {
	return &TracingBlockstore{
		tracedMu:   sync.Mutex{},
		traced:     make(map[cid.Cid]struct{}),
		Blockstore: bs,
	}
}

var (
	_ cbor.IpldBlockstore   = (*TracingBlockstore)(nil)
	_ blockstore.Viewer     = (*TracingBlockstore)(nil)
	_ blockstore.Blockstore = (*TracingBlockstore)(nil)
)

type TracingBlockstore struct {
	api impl.FullNodeAPI

	tracedMu sync.Mutex
	traced   map[cid.Cid]struct{}

	blockstore.Blockstore
}

func (tb *TracingBlockstore) Traced() map[cid.Cid]struct{} {
	// TODO maybe returning a copy?
	return tb.traced
}

// implements blockstore viewer interface.
func (tb *TracingBlockstore) View(k cid.Cid, callback func([]byte) error) error {
	blk, err := tb.Get(k)
	if err == nil && blk != nil {
		return callback(blk.RawData())
	}
	return err
}

func (tb *TracingBlockstore) Get(cid cid.Cid) (blocks.Block, error) {
	tb.tracedMu.Lock()
	tb.traced[cid] = struct{}{}
	tb.tracedMu.Unlock()

	block, err := tb.Blockstore.Get(cid)
	if err != nil {
		return nil, err
	}
	return block, err
}

func (tb *TracingBlockstore) Put(block blocks.Block) error {
	tb.tracedMu.Lock()
	tb.traced[block.Cid()] = struct{}{}
	tb.tracedMu.Unlock()
	return tb.Blockstore.Put(block)
}

func (tb *TracingBlockstore) PutMany(blocks []blocks.Block) error {
	tb.tracedMu.Lock()
	for _, b := range blocks {
		tb.traced[b.Cid()] = struct{}{}
	}
	tb.tracedMu.Unlock()
	return tb.Blockstore.PutMany(blocks)
}
