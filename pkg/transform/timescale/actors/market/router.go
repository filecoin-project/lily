package market

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v0"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v9"
)

type MarketHandler = func(ctx context.Context, s store.Store, current, executed *types.TipSet, marketRoot cid.Cid) (model.PersistableList, error)

func MakeMarketProcessor(av actortypes.Version) (MarketHandler, error) {
	switch av {
	case actortypes.Version0:
		return v0.MarketHandler, nil
	case actortypes.Version9:
		return v9.MarketHandler, nil
	default:
		return nil, fmt.Errorf("unsupported market actor version: %d", av)
	}
}
