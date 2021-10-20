package tasks

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
)

var _ adt.Store = (*TaskStore)(nil)

type TaskStore struct {
	cache *lru.ARCCache
	store blockstore.Blockstore
}

func (t *TaskStore) Context() context.Context {
	return context.TODO()
}

func NewTaskStore(chainBs, stateBs blockstore.Blockstore, cacheSize int) *TaskStore {
	cache, err := lru.NewARC(cacheSize)
	if err != nil {
		panic(err) // only errors if lru.New is given a neg value
	}
	return &TaskStore{cache: cache, store: blockstore.Union(chainBs, stateBs)}
}

func (t *TaskStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	cu, ok := out.(cbg.CBORUnmarshaler)
	if !ok {
		return xerrors.Errorf("out parameter does not implement CBORUnmarshaler")
	}
	if raw, hit := t.cache.Get(c); hit {
		return cu.UnmarshalCBOR(bytes.NewReader(raw.(blocks.Block).RawData()))
	}
	// cache miss
	blk, err := t.store.Get(c)
	if err != nil {
		return err
	}
	// cache store
	t.cache.Add(c, blk)
	return cu.UnmarshalCBOR(bytes.NewReader(blk.RawData()))
}

func (t *TaskStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	// new store operation is cheap-ish
	return cbornode.NewCborStore(t.store).Put(ctx, v)
}

func (t *TaskStore) Warm(ts *types.TipSet) {
	// TODO something like this, we want to load as much of the tree into cache as we can.
	tree, err := state.LoadStateTree(t, ts.ParentState())
	if err != nil {
		panic(err)
	}
	if err := tree.ForEach(func(address address.Address, actor *types.Actor) error {
		return nil
	}); err != nil {
		panic(err)
	}
}
