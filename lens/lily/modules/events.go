package modules

import (
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"go.uber.org/fx"

	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/storage"
)

func NewEvents(mctx helpers.MetricsCtx, lc fx.Lifecycle, chainAPI full.ChainModuleAPI, stateAPI full.StateModuleAPI) (*events.Events, error) {
	api := struct {
		full.ChainModuleAPI
		full.StateModuleAPI
	}{
		ChainModuleAPI: chainAPI,
		StateModuleAPI: stateAPI,
	}

	return events.NewEventsWithConfidence(mctx, api, 10)
}

func NewStorageCatalog(mctx helpers.MetricsCtx, lc fx.Lifecycle, cfg *config.Conf) (*storage.Catalog, error) {
	return storage.NewCatalog(cfg.Storage)
}

func LoadConf(path string) func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*config.Conf, error) {
	return func(mctx helpers.MetricsCtx, lc fx.Lifecycle) (*config.Conf, error) {
		return config.FromFile(path)
	}
}
