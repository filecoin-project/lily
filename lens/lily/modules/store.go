package modules

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/util"
)

func CacheConfig(blockstoreCacheSize int, statestoreCacheSize int) func(lc fx.Lifecycle, mctx helpers.MetricsCtx) (*util.CacheConfig, error) {
	return func(lc fx.Lifecycle, mctx helpers.MetricsCtx) (*util.CacheConfig, error) {
		return &util.CacheConfig{
			BlockstoreCacheSize: blockstoreCacheSize,
			StatestoreCacheSize: statestoreCacheSize,
		}, nil
	}
}

func CachingUniversalBlockstore(cacheSize int) func(lc fx.Lifecycle, mctx helpers.MetricsCtx, r repo.LockedRepo) (dtypes.UniversalBlockstore, error) {
	return func(lc fx.Lifecycle, mctx helpers.MetricsCtx, r repo.LockedRepo) (dtypes.UniversalBlockstore, error) {
		bs, err := r.Blockstore(helpers.LifecycleCtx(mctx, lc), repo.UniversalBlockstore)
		if err != nil {
			return nil, err
		}
		if c, ok := bs.(io.Closer); ok {
			lc.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					return c.Close()
				},
			})
		}

		if cacheSize <= 0 {
			return nil, xerrors.Errorf("invalid value for blockstore cache size: must be a positive integer")
		}

		cbs, err := util.NewCachingBlockstore(bs, cacheSize)
		if err != nil {
			return nil, xerrors.Errorf("failed to create caching blockstore: %v", err)
		}

		return cbs, err
	}
}
