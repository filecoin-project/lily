package chaineconomics

import (
	"context"
	"github.com/filecoin-project/lily/model"
	chainmodel "github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lotus/chain/types"
)

func init() {
	model.RegisterTipSetModelExtractor(&chainmodel.ChainEconomics{}, ChainEconomicsExtractor{})
}

var _ model.TipSetStateExtractor = (*ChainEconomicsExtractor)(nil)

type ChainEconomicsExtractor struct{}

func (ChainEconomicsExtractor) Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error) {
	return ExtractChainEconomicsModel(ctx, api, current)
}

func (ChainEconomicsExtractor) Name() string {
	return "chain_economics"
}
