package market

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	v1 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func TransformMarketState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root *cid.Cid) (model.Persistable, error) {
	if root == nil {
		return model.PersistableList{
			data.StartProcessingReport(tasktype.MarketDealState, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.MarketDealProposal, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
		}, nil
	}
	switch version {
	case actortypes.Version0,
		actortypes.Version2,
		actortypes.Version3,
		actortypes.Version4,
		actortypes.Version5,
		actortypes.Version6,
		actortypes.Version7,
		actortypes.Version8,
		actortypes.Version9:
		return v1.TransformMarketState(ctx, s, version, current, executed, *root)
	}
	return nil, fmt.Errorf("unsupported version : %d", version)

}
