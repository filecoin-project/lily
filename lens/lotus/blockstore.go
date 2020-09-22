package lotus

import (
	"context"
	"io"
	"path/filepath"

	"golang.org/x/xerrors"

	dgbadger "github.com/dgraph-io/badger/v2"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/lib/blockstore"
	cid "github.com/ipfs/go-cid"
	badger "github.com/ipfs/go-ds-badger2"
)

// BlockstoreBackedAPI is a wrapper around a full node api that bypasses the api for reading block data and
// gets it directly from the blockstore
type BlockstoreBackedAPI struct {
	api.FullNode
	bs blockstore.Blockstore
}

type closer func() error

func (c closer) Close() error { return c() }

func NewBlockstoreBackedAPI(path string, api api.FullNode) (*BlockstoreBackedAPI, io.Closer, error) {

	chainPath := filepath.Join(path, "datastore", "chain")
	ds, err := openChainDs(chainPath)
	if err != nil {
		return nil, nil, xerrors.Errorf("open chain datastore: %w", err)
	}

	bs := blockstore.NewBlockstore(ds)

	return &BlockstoreBackedAPI{
		FullNode: api,
		bs:       bs,
	}, closer(ds.Close), nil

}

func openChainDs(path string) (*badger.Datastore, error) {
	opts := badger.DefaultOptions
	opts.GcInterval = 0 // disable GC for chain datastore

	opts.Options = dgbadger.DefaultOptions("").
		WithTruncate(true).
		WithValueThreshold(1 << 10).
		WithReadOnly(true)

	return badger.NewDatastore(path, &opts)
}

func (s *BlockstoreBackedAPI) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	blk, err := s.bs.Get(obj)
	if err != nil {
		return nil, xerrors.Errorf("blockstore get: %w", err)
	}

	return blk.RawData(), nil
}
