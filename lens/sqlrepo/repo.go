package sqlrepo

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	pgchainbs "github.com/filecoin-project/go-bs-postgres-chainnotated"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	pgbsCfg := pgchainbs.PgBlockstoreConfig{
		PgxConnectString:          c.String("repo"),
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

	return util.NewAPIOpener(c.Context, pgbs, getHeadWithOffset, c.Int("lens-cache-hint"))
}
