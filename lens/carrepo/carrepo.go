package carrepo

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/ulimit"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/willscott/carbs"

	"github.com/filecoin-project/sentinel-visor/lens/util"

	"github.com/filecoin-project/sentinel-visor/lens"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	if _, _, err := ulimit.ManageFdLimit(); err != nil {
		return nil, nil, fmt.Errorf("setting file descriptor limit: %s", err)
	}

	db, err := carbs.Load(c.String("lens-repo"), false)
	if err != nil {
		return nil, nil, err
	}
	cacheDB := util.NewCachingStore(&wrapper{c: db})

	h := func(ctx context.Context, lookback int) (*types.TipSetKey, error) {
		c, err := db.Roots()
		if err != nil {
			return nil, err
		}
		tsk := types.NewTipSetKey(c...)
		return &tsk, nil
	}

	return util.NewAPIOpener(c.Context, cacheDB, h, c.Int("lens-cache-hint"))
}

type wrapper struct {
	c *carbs.Carbs
}

func (w *wrapper) View(c cid.Cid, callback func([]byte) error) error {
	blk, err := w.Get(c)
	if err != nil {
		return err
	}
	return callback(blk.RawData())
}

func (w *wrapper) DeleteMany(cids []cid.Cid) error {
	for _, c := range cids {
		if err := w.c.DeleteBlock(c); err != nil {
			return err
		}
	}
	return nil
}

func (w *wrapper) DeleteBlock(c cid.Cid) error {
	return w.c.DeleteBlock(c)
}

func (w *wrapper) Has(c cid.Cid) (bool, error) {
	return w.c.Has(c)
}

func (w *wrapper) Get(c cid.Cid) (blocks.Block, error) {
	return w.c.Get(c)
}

func (w *wrapper) GetSize(c cid.Cid) (int, error) {
	return w.c.GetSize(c)
}

func (w *wrapper) Put(blk blocks.Block) error {
	return w.c.Put(blk)
}

func (w *wrapper) PutMany(blks []blocks.Block) error {
	return w.c.PutMany(blks)
}

func (w *wrapper) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return w.c.AllKeysChan(ctx)
}

func (w *wrapper) HashOnRead(enabled bool) {
	w.c.HashOnRead(enabled)
}
