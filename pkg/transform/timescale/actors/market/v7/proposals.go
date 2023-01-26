package v7

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/market"

	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/pkg/core"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v7"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/util"
)

type Proposals struct{}

func (Proposals) Transform(ctx context.Context, current, executed *types.TipSet, change *marketdiff.StateDiffResult) (model.Persistable, error) {
	var marketProposals []*proposal
	for _, change := range change.DealProposalChanges {
		// we only car about new and modified deal proposals
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		dealProp := new(market.DealProposal)
		if err := dealProp.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			return nil, err
		}
		marketProposals = append(marketProposals, &proposal{
			DealID: change.DealID,
			State:  dealProp,
		})
	}
	return MarketDealProposalChangesAsModel(ctx, current, marketProposals)
}

type proposal struct {
	DealID uint64
	State  *market.DealProposal
}

func MarketDealProposalChangesAsModel(ctx context.Context, current *types.TipSet, dealProps []*proposal) (model.Persistable, error) {
	dealPropsModel := make(marketmodel.MarketDealProposals, len(dealProps))
	for i, prop := range dealProps {
		label := base64.StdEncoding.EncodeToString([]byte(util.SanitizeLabel(prop.State.Label)))
		dealPropsModel[i] = &marketmodel.MarketDealProposal{
			Height:               int64(current.Height()),
			DealID:               prop.DealID,
			StateRoot:            current.ParentState().String(),
			PaddedPieceSize:      uint64(prop.State.PieceSize),
			UnpaddedPieceSize:    uint64(prop.State.PieceSize.Unpadded()),
			StartEpoch:           int64(prop.State.StartEpoch),
			EndEpoch:             int64(prop.State.EndEpoch),
			ClientID:             prop.State.Client.String(),
			ProviderID:           prop.State.Provider.String(),
			ClientCollateral:     prop.State.ClientCollateral.String(),
			ProviderCollateral:   prop.State.ProviderCollateral.String(),
			StoragePricePerEpoch: prop.State.StoragePricePerEpoch.String(),
			PieceCID:             prop.State.PieceCID.String(),
			IsVerified:           prop.State.VerifiedDeal,
			Label:                label,
			IsString:             false,
		}
	}
	return dealPropsModel, nil
}
