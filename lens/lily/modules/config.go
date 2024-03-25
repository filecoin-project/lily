package modules

import (
	"go.uber.org/fx"

	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/storage"

	"github.com/filecoin-project/lotus/node/modules/helpers"
)

func NewStorageCatalog(_ helpers.MetricsCtx, _ fx.Lifecycle, cfg *config.Conf) (*storage.Catalog, error) {
	return storage.NewCatalog(cfg.Storage)
}

func LoadConf(path string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*config.Conf, error) {
	return func(_ helpers.MetricsCtx, _ fx.Lifecycle) (*config.Conf, error) {
		return config.FromFile(path)
	}
}

func NewQueueCatalog(_ helpers.MetricsCtx, _ fx.Lifecycle, cfg *config.Conf) (*distributed.Catalog, error) {
	return distributed.NewCatalog(cfg.Queue)
}
