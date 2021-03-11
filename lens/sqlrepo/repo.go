package sqlrepo

import (
	"context"
	"fmt"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"

	pgchainbs "github.com/filecoin-project/go-bs-postgres-chainnotated"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	pgbsCfg := pgchainbs.PgBlockstoreConfig{
		PgxConnectString:          c.String("lens-repo"),
		InstanceNamespace:         c.String("lens-postgres-namespace"),
		CachePreloadRecentBlocks:  c.Bool("lens-postgres-preload-recents"),
		PrefetchDagLayersOnDbRead: int32(c.Int("lens-postgres-get-prefetch-depth")),
		CacheInactiveBeforeRead:   true,
		DisableBlocklinkParsing:   true,
		LogCacheStatsOnUSR1:       true,
	}

	if customCacheSize := c.Int("lens-cache-hint"); customCacheSize != 1024*1024 {
		pgbsCfg.CacheSizeGiB = uint64(customCacheSize)
	}

	pgbs, err := pgchainbs.NewPgBlockstore(c.Context, pgbsCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("blockstore instantiation failed: %w", err)
	}

	var getHeadWithOffset util.HeadMthd = func(ctx context.Context, lookback int) (*types.TipSetKey, error) {
		tsd, err := pgbs.GetFilTipSetHead(ctx)
		if err != nil {
			return nil, err
		}

		if lookback != 0 {
			tsd, err = pgbs.FindFilTipSet(ctx, tsd.TipSetCids, abi.ChainEpoch(lookback))
			if err != nil {
				return nil, err
			}
		}

		tsk := types.NewTipSetKey(tsd.TipSetCids...)
		return &tsk, nil
	}

	return util.NewAPIOpener(c.Context, &wrapper{pg: pgbs}, getHeadWithOffset, c.Int("lens-cache-hint"))
}

type wrapper struct {
	pg *pgchainbs.PgBlockstore
}

func (w *wrapper) DeleteBlock(c cid.Cid) error {
	return w.pg.DeleteBlock(c)
}

func (w *wrapper) Has(c cid.Cid) (bool, error) {
	return w.pg.Has(c)
}

func (w *wrapper) Get(c cid.Cid) (blocks.Block, error) {
	return w.pg.Get(c)
}

func (w *wrapper) GetSize(c cid.Cid) (int, error) {
	return w.pg.GetSize(c)
}

func (w *wrapper) Put(blk blocks.Block) error {
	return w.pg.Put(blk)
}

func (w *wrapper) PutMany(blks []blocks.Block) error {
	return w.pg.PutMany(blks)
}

func (w *wrapper) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return w.pg.AllKeysChan(ctx)
}

func (w *wrapper) HashOnRead(enabled bool) {
	w.pg.HashOnRead(enabled)
}

func (w *wrapper) View(c cid.Cid, callback func([]byte) error) error {
	return w.pg.View(c, callback)
}

func (w *wrapper) DeleteMany(cids []cid.Cid) error {
	for _, c := range cids {
		if err := w.pg.DeleteBlock(c); err != nil {
			return err
		}
	}
	return nil
}
