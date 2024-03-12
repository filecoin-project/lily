package modules

import (
	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/modules/helpers"
)

func NewEvents(mctx helpers.MetricsCtx, _ fx.Lifecycle, chainAPI full.ChainModuleAPI, stateAPI full.StateModuleAPI) (*events.Events, error) {
	api := struct {
		full.ChainModuleAPI
		full.StateModuleAPI
	}{
		ChainModuleAPI: chainAPI,
		StateModuleAPI: stateAPI,
	}

	return events.NewEvents(mctx, api)
}
