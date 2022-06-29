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

	currDealProposals, err := ec.CurrState.Proposals()
	if err != nil {
		return nil, fmt.Errorf("loading current market deal proposals: %w", err)
	}

	if ec.IsGenesis() {
		var out marketmodel.MarketDealProposals
		if err := currDealProposals.ForEach(func(id abi.DealID, dp market.DealProposal) error {
			var label string
			if dp.Label.IsString() {
				var err error
				label, err = dp.Label.ToString()
				if err != nil {
					return fmt.Errorf("creating deal proposal label string: %w", err)
				}
			} else {
				label = ""
			}
			out = append(out, &marketmodel.MarketDealProposal{
				Height:               int64(ec.CurrTs.Height()),
				DealID:               uint64(id),
				StateRoot:            ec.CurrTs.ParentState().String(),
				PaddedPieceSize:      uint64(dp.PieceSize),
				UnpaddedPieceSize:    uint64(dp.PieceSize.Unpadded()),
				StartEpoch:           int64(dp.StartEpoch),
				EndEpoch:             int64(dp.EndEpoch),
				ClientID:             dp.Client.String(),
				ProviderID:           dp.Provider.String(),
				ClientCollateral:     dp.ClientCollateral.String(),
				ProviderCollateral:   dp.ProviderCollateral.String(),
				StoragePricePerEpoch: dp.StoragePricePerEpoch.String(),
				PieceCID:             dp.PieceCID.String(),
				IsVerified:           dp.VerifiedDeal,
				Label:                SanitizeLabel(label),
			})
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walking current deal states: %w", err)
		}
		return out, nil

	}

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

	out := make(marketmodel.MarketDealProposals, len(changes.Added))
	for idx, add := range changes.Added {
		var label string
		if add.Proposal.Label.IsString() {
			var err error
			label, err = add.Proposal.Label.ToString()
			if err != nil {
				return nil, fmt.Errorf("creating deal proposal label string: %w", err)
			}
		} else {
			label = ""
		}
		out[idx] = &marketmodel.MarketDealProposal{
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
