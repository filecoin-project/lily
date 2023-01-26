package v7

import (
	"bytes"
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/market"

	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/pkg/core"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v7"
)

type Deals struct{}

func (Deals) Transform(ctx context.Context, current, executed *types.TipSet, changes *marketdiff.StateDiffResult) (model.Persistable, error) {
	var marketDeals []*deals
	for _, change := range changes.DealStateChanges {
		// only care about new and modified deal states
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		dealState := new(market.DealState)
		if err := dealState.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			return nil, err
		}
		marketDeals = append(marketDeals, &deals{
			DealID: change.DealID,
			State:  dealState,
		})
	}
	return MarketDealStateChangesAsModel(ctx, current, marketDeals)
}

type deals struct {
	DealID uint64
	State  *market.DealState
}

func MarketDealStateChangesAsModel(ctx context.Context, current *types.TipSet, dealStates []*deals) (model.Persistable, error) {
	dealStateModel := make(marketmodel.MarketDealStates, len(dealStates))
	for i, deal := range dealStates {
		dealStateModel[i] = &marketmodel.MarketDealState{
			Height:           int64(current.Height()),
			StateRoot:        current.ParentState().String(),
			DealID:           deal.DealID,
			SectorStartEpoch: int64(deal.State.SectorStartEpoch),
			LastUpdateEpoch:  int64(deal.State.LastUpdatedEpoch),
			SlashEpoch:       int64(deal.State.SlashEpoch),
		}
	}
	return dealStateModel, nil
}
