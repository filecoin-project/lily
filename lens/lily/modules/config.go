package modules

import (
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"go.uber.org/fx"

	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/queue"
	"github.com/filecoin-project/lily/storage"
)

func NewStorageCatalog(mctx helpers.MetricsCtx, lc fx.Lifecycle, cfg *config.Conf) (*storage.Catalog, error) {
	return storage.NewCatalog(cfg.Storage)
}

func NewQueueCatalog(mctx helpers.MetricsCtx, lc fx.Lifecycle, cfg *config.Conf) (*queue.Catalog, error) {
	return queue.NewCatalog(cfg.Queue)
}

func LoadConf(path string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*config.Conf, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*config.Conf, error) {
		return config.FromFile(path)
	}
}
