package v8

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/pkg/core"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/util"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

type Proposals struct{}

func (Proposals) Transform(ctx context.Context, current, executed *types.TipSet, change *marketdiff.StateDiffResult) model.Persistable {
	report := data.StartProcessingReport(tasktype.MarketDealProposal, current)
	var marketProposals []*proposal
	for _, change := range change.DealProposalChanges {
		// we only car about new and modified deal proposals
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		dealProp := new(market.DealProposal)
		if err := dealProp.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			report.AddError(err)
			continue
		}
		marketProposals = append(marketProposals, &proposal{
			DealID: change.DealID,
			State:  dealProp,
		})
	}
	m, err := MarketDealProposalChangesAsModel(ctx, current, marketProposals)
	if err != nil {
		return report.AddError(err).Finish()
	}
	return report.AddModels(m).Finish()
}

type proposal struct {
	DealID uint64
	State  *market.DealProposal
}

func MarketDealProposalChangesAsModel(ctx context.Context, current *types.TipSet, dealProps []*proposal) (model.Persistable, error) {
	dealPropsModel := make(marketmodel.MarketDealProposals, len(dealProps))
	for i, prop := range dealProps {
		var isString bool
		var base64Label string
		if prop.State.Label.IsString() {
			labelString, err := prop.State.Label.ToString()
			if err != nil {
				return nil, fmt.Errorf("deal proposal (ID: %d) label is not a string despite claiming it is (developer error?)", prop.DealID)
			}

			isString = true
			base64Label = base64.StdEncoding.EncodeToString([]byte(util.SanitizeLabel(labelString)))

		} else if prop.State.Label.IsBytes() {
			labelBytes, err := prop.State.Label.ToBytes()
			if err != nil {
				return nil, fmt.Errorf("deal proposal (ID: %d) label is not bytes despit claiming it is (developer error?)", prop.DealID)
			}

			isString = false
			base64Label = base64.StdEncoding.EncodeToString(labelBytes)

		} else {
			// TODO this should never happen, but if it does it indicates a logic.
			return nil, fmt.Errorf("deal proposal (ID: %d) label is neither bytes nor string (DEVELOPER ERROR)", prop.DealID)
		}
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
			Label:                base64Label,
			IsString:             isString,
		}
	}
	return dealPropsModel, nil
}
