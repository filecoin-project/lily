package modules

import (
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/lotus/node/modules/helpers"
	"go.uber.org/fx"
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
