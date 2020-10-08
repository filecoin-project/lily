package actorstate

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	market "github.com/filecoin-project/lotus/chain/actors/builtin/market"
	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	marketmodel "github.com/filecoin-project/sentinel-visor/model/actors/market"
)

// was services/processor/tasks/market/market.go

// StorageMarketExtractor extracts market actor state
type StorageMarketExtractor struct{}

func init() {
	Register(builtin.StorageMarketActorCodeID, StorageMarketExtractor{})
}

func (m StorageMarketExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMarketExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	proposals, err := m.marketDealProposalChanges(ctx, a, node)
	if err != nil {
		return nil, err
	}

	states, err := m.marketDealStateChanges(ctx, a, node)
	if err != nil {
		return nil, err
	}

	return &marketmodel.MarketTaskResult{
		Proposals: proposals,
		States:    states,
	}, nil
}

func (m StorageMarketExtractor) marketDealStateChanges(ctx context.Context, a ActorInfo, node ActorStateAPI) (marketmodel.MarketDealStates, error) {
	// TODO: pass in diff to avoid doing it twice
	pred := state.NewStatePredicates(node)
	stateDiff := pred.OnStorageMarketActorChanged(pred.OnDealStateChanged(pred.OnDealStateAmtChanged()))
	changed, val, err := stateDiff(ctx, a.ParentTipSet, a.TipSet)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, xerrors.Errorf("no state change detected")
	}
	changes, ok := val.(*market.DealStateChanges)
	if !ok {
		// indicates a developer error or breaking change in lotus
		return nil, xerrors.Errorf("Unknown type returned by Deal State AMT predicate: %T", val)
	}

	out := make(marketmodel.MarketDealStates, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealState{
			DealID:           uint64(add.ID),
			StateRoot:        a.ParentStateRoot.String(),
			SectorStartEpoch: int64(add.Deal.SectorStartEpoch),
			LastUpdateEpoch:  int64(add.Deal.LastUpdatedEpoch),
			SlashEpoch:       int64(add.Deal.SlashEpoch),
		}
		idx++
	}
	for _, mod := range changes.Modified {
		out[idx] = &marketmodel.MarketDealState{
			DealID:           uint64(mod.ID),
			SectorStartEpoch: int64(mod.To.SectorStartEpoch),
			LastUpdateEpoch:  int64(mod.To.LastUpdatedEpoch),
			SlashEpoch:       int64(mod.To.SlashEpoch),
			StateRoot:        a.ParentStateRoot.String(),
		}
		idx++
	}
	return out, nil
}

func (m StorageMarketExtractor) marketDealProposalChanges(ctx context.Context, a ActorInfo, node ActorStateAPI) (marketmodel.MarketDealProposals, error) {
	// TODO: pass in diff to avoid doing it twice
	pred := state.NewStatePredicates(node)
	stateDiff := pred.OnStorageMarketActorChanged(pred.OnDealProposalChanged(pred.OnDealProposalAmtChanged()))
	changed, val, err := stateDiff(ctx, a.ParentTipSet, a.TipSet)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, nil
	}
	changes, ok := val.(*market.DealProposalChanges)
	if !ok {
		// indicates a developer error or breaking change in lotus
		return nil, xerrors.Errorf("Unknown type returned by Deal Proposal AMT predicate: %T", val)
	}

	out := make(marketmodel.MarketDealProposals, len(changes.Added))

	for idx, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealProposal{
			DealID:               uint64(add.ID),
			StateRoot:            a.ParentStateRoot.String(),
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
			Label:                add.Proposal.Label,
		}
	}
	return out, nil
}
