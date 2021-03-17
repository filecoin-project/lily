package sqlrepo

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	pgchainbs "github.com/filecoin-project/go-bs-postgres-chainnotated"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/util"
)

func NewAPIOpener(c *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	pgbsCfg := pgchainbs.PgBlockstoreConfig{
		PgxConnectString:        c.String("repo"),
		CacheInactiveBeforeRead: true,
		DisableBlocklinkParsing: true,
		LogCacheStatsOnUSR1:     true,
	}

	enablePreload := os.Getenv("LOTUS_CHAINSTORE_PRELOAD_RECENTS")
	if enablePreload != "" && enablePreload != "0" && enablePreload != "false" {
		pgbsCfg.CachePreloadRecentBlocks = true
	}

	pgbsCfg.InstanceNamespace = c.String("lens-postgres-namespace")
	if pgbsCfg.InstanceNamespace == "" {
		pgbsCfg.InstanceNamespace = "main"
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
