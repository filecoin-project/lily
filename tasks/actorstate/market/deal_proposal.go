package market

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/market")

var _ actorstate.ActorStateExtractor = (*DealProposalExtractor)(nil)

type DealProposalExtractor struct{}

func (DealProposalExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "DealProposalExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "DealProposalExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMarketStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	var dealProposals []market.ProposalIDState
	// if this is genesis iterator actors current state.
	if ec.IsGenesis() {
		currDealProposals, err := ec.CurrState.Proposals()
		if err != nil {
			return nil, fmt.Errorf("loading current market deal proposals: %w", err)
		}

		if err := currDealProposals.ForEach(func(id abi.DealID, dp market.DealProposal) error {
			dealProposals = append(dealProposals, market.ProposalIDState{
				ID:       id,
				Proposal: dp,
			})
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		// else diff the actor against previous state and collect any additions that occurred.
		changed, err := ec.CurrState.ProposalsChanged(ec.PrevState)
		if err != nil {
			return nil, fmt.Errorf("checking for deal proposal changes: %w", err)
		}
		if !changed {
			return nil, nil
		}

		changes, err := market.DiffDealProposals(ctx, ec.Store, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, fmt.Errorf("diffing deal states: %w", err)
		}

		for _, change := range changes.Added {
			dealProposals = append(dealProposals, market.ProposalIDState{
				ID:       change.ID,
				Proposal: change.Proposal,
			})
		}
	}

	out := make(marketmodel.MarketDealProposalsV1_7, len(dealProposals))
	for idx, add := range dealProposals {
		// DealProposalExtractor is only called for market actors v7 and prior. If the label is not a valid UTF-8 string
		// retrieve it as bytes and pass the bytes as a string to SanitizeLabel which mimics behaviour prior to this change.
		var label string
		if add.Proposal.Label.IsString() {
			var err error
			label, err = add.Proposal.Label.ToString()
			if err != nil {
				return nil, fmt.Errorf("deal proposal (ID: %d) label is not a string dispite claiming it is (developer error?)", add.ID)
			}
		} else if add.Proposal.Label.IsBytes() {
			dealBytes, err := add.Proposal.Label.ToBytes()
			if err != nil {
				return nil, fmt.Errorf("deal proposal (ID: %d) label is not bytes dispite claiming it is (developer error?)", add.ID)
			}
			label = string(dealBytes)
		} else {
			// TODO this should never happen, but if it does it indicates a bug in the actor code
			return nil, fmt.Errorf("deal proposal (ID: %d) label is neither bytes nor string (DEVELOPER ERROR)", add.ID)
		}
		out[idx] = &marketmodel.MarketDealProposalV1_7{
			Height:               int64(ec.CurrTs.Height()),
			DealID:               uint64(add.ID),
			StateRoot:            ec.CurrTs.ParentState().String(),
			PaddedPieceSize:      uint64(add.Proposal.PieceSize),
			UnpaddedPieceSize:    uint64(add.Proposal.PieceSize.Unpadded()),
			StartEpoch:           int64(add.Proposal.StartEpoch),
			EndEpoch:             int64(add.Proposal.EndEpoch),
			ClientID:             add.Proposal.Client.String(),
			ProviderID:           add.Proposal.Provider.String(),
			ClientCollateral:     add.Proposal.ClientCollateral.String(),
			ProviderCollateral:   add.Proposal.ProviderCollateral.String(),
			StoragePricePerEpoch: add.Proposal.StoragePricePerEpoch.String(),
			PieceCID:             add.Proposal.PieceCID.String(),
			IsVerified:           add.Proposal.VerifiedDeal,
			Label:                SanitizeLabel(label),
		}
	}
	return out, nil
}
