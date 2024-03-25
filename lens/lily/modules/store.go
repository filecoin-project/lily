package modules

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/fx"

	"github.com/filecoin-project/lily/lens/util"

	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"github.com/filecoin-project/lotus/node/repo"
)

func CacheConfig(blockstoreCacheSize uint, statestoreCacheSize uint) func(_ fx.Lifecycle, mctx helpers.MetricsCtx) (*util.CacheConfig, error) {
	return func(_ fx.Lifecycle, _ helpers.MetricsCtx) (*util.CacheConfig, error) {
		return &util.CacheConfig{
			BlockstoreCacheSize: blockstoreCacheSize,
			StatestoreCacheSize: statestoreCacheSize,
		}, nil
	}
}

func NewCachingUniversalBlockstore(lc fx.Lifecycle, mctx helpers.MetricsCtx, cc *util.CacheConfig, r repo.LockedRepo) (dtypes.UniversalBlockstore, error) {
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

	if cc.BlockstoreCacheSize == 0 {
		return bs, nil
	}

	log.Infof("creating caching blockstore with size=%d", cc.BlockstoreCacheSize)
	cbs, err := util.NewCachingBlockstore(bs, int(cc.BlockstoreCacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create caching blockstore: %v", err)
	}

	return cbs, err
}
