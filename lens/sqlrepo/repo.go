package sqlrepo

import (
	"context"
	"fmt"
	"os"
	"strconv"

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

	envVarCacheGiB := "LOTUS_CHAINSTORE_CACHE_GIB"
	envVarNamespace := "LOTUS_CHAINSTORE_SCHEMA_NAMESPACE"

	pgbsCfg.InstanceNamespace = os.Getenv(envVarNamespace)
	if pgbsCfg.InstanceNamespace == "" {
		pgbsCfg.InstanceNamespace = "main"
	}

	if customCacheSizeStr := os.Getenv(envVarCacheGiB); customCacheSizeStr != "" {
		var parseErr error
		if pgbsCfg.CacheSizeGiB, parseErr = strconv.ParseUint(customCacheSizeStr, 10, 8); parseErr != nil {
			return nil, nil, fmt.Errorf("failed to parse requested cache size '%s': %w", customCacheSizeStr, parseErr)
		}
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
