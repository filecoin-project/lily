package s3repo

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

var log = logging.Logger("sql")

type S3Blockstore struct {
	prefix string
	client http.Client
}

func NewBlockStore(connstr string) (blockstore.Blockstore, error) {
	sbs := &S3Blockstore{
		prefix: connstr,
		client: http.Client{},
	}

	// we do not currently use the Identity codec, but just in case...
	return blockstore.WrapIDStore(sbs), nil
}

func (sbs *S3Blockstore) Has(c cid.Cid) (has bool, err error) {
	resp, err := sbs.client.Head(sbs.prefix + c.String() + "/data.raw")
	if err != nil {
		return false, err
	}
	return resp.StatusCode == 200, nil
}

func (sbs *S3Blockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, fmt.Errorf("not implemented")
}

func (sbs *S3Blockstore) Get(c cid.Cid) (blocks.Block, error) {
	resp, err := sbs.client.Get(sbs.prefix + c.String() + "/data.raw")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch: %v", resp.StatusCode)
	}
	defer resp.Body.Close() // nolint: errcheck
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return blocks.NewBlockWithCid(buf, c)
}

func (sbs *S3Blockstore) GetSize(c cid.Cid) (size int, err error) {
	resp, err := sbs.client.Head(sbs.prefix + c.String() + "/data.raw")
	if err != nil {
		return -1, err
	}
	if resp.StatusCode == 200 {
		return int(resp.ContentLength), nil
	}
	return -1, fmt.Errorf("does not exist")
}

func (sbs *S3Blockstore) getMasterTsKey(ctx context.Context, lookback int) (*types.TipSetKey, error) {
	resp, err := sbs.client.Get(sbs.prefix + "/head")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint: errcheck
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cidStrs := strings.Split(string(buf), " ")
	cids := make([]cid.Cid, len(cidStrs))
	for _, cs := range cidStrs {
		c, err := cid.Parse(cs)
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	tk := types.NewTipSetKey(cids...)
	return &tk, nil
}

// BEGIN UNIMPLEMENTED

// HashOnRead specifies if every read block should be
// rehashed to make sure it matches its CID.
func (sbs *S3Blockstore) HashOnRead(enabled bool) {
	log.Warn("HashOnRead toggle not implemented, ignoring")
}

// Put puts a given block to the underlying datastore
func (sbs *S3Blockstore) Put(b blocks.Block) (err error) {
	return fmt.Errorf("not implemented")
}

func (sbs *S3Blockstore) PutMany(blks []blocks.Block) error {
	return fmt.Errorf("not implemented")
}

func (sbs *S3Blockstore) DeleteBlock(cid.Cid) error {
	return fmt.Errorf("not implemented")
}
