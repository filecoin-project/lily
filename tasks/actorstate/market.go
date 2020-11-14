package actorstate

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	market "github.com/filecoin-project/lotus/chain/actors/builtin/market"
	"github.com/filecoin-project/lotus/chain/events/state"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	marketmodel "github.com/filecoin-project/sentinel-visor/model/actors/market"
)

// was services/processor/tasks/market/market.go

// StorageMarketExtractor extracts market actor state
type StorageMarketExtractor struct{}

func init() {
	Register(sa0builtin.StorageMarketActorCodeID, StorageMarketExtractor{})
	Register(sa2builtin.StorageMarketActorCodeID, StorageMarketExtractor{})
}

func (m StorageMarketExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.PersistableWithTx, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMarketExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	pred := state.NewStatePredicates(node)
	stateDiff := pred.OnStorageMarketActorChanged(storageMarketChangesPred(pred))
	changed, val, err := stateDiff(ctx, a.ParentTipSet, a.TipSet)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, xerrors.Errorf("no state change detected")
	}

	mchanges, ok := val.(*marketChanges)
	if !ok {
		return nil, xerrors.Errorf("Unknown type returned by market changes predicate: %T", val)
	}

	res := &marketmodel.MarketTaskResult{}

	if mchanges.ProposalChanges != nil {
		proposals, err := m.marketDealProposalChanges(ctx, a, mchanges.ProposalChanges)
		if err != nil {
			return nil, err
		}
		res.Proposals = proposals
	}

	if mchanges.DealChanges != nil {
		states, err := m.marketDealStateChanges(ctx, a, mchanges.DealChanges)
		if err != nil {
			return nil, err
		}
		res.States = states
	}

	return res, nil
}

func (m StorageMarketExtractor) marketDealStateChanges(ctx context.Context, a ActorInfo, changes *market.DealStateChanges) (marketmodel.MarketDealStates, error) {
	out := make(marketmodel.MarketDealStates, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealState{
			Height:           int64(a.Epoch),
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
			Height:           int64(a.Epoch),
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

func (m StorageMarketExtractor) marketDealProposalChanges(ctx context.Context, a ActorInfo, changes *market.DealProposalChanges) (marketmodel.MarketDealProposals, error) {
	out := make(marketmodel.MarketDealProposals, len(changes.Added))

	for idx, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealProposal{
			Height:               int64(a.Epoch),
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

type marketChanges struct {
	DealChanges     *market.DealStateChanges
	ProposalChanges *market.DealProposalChanges
}

// storageMarketChangesPred returns a DiffStorageMarketStateFunc that extracts deal state and deal proposal changes from
// a single state change.
func storageMarketChangesPred(pred *state.StatePredicates) state.DiffStorageMarketStateFunc {
	return func(ctx context.Context, oldState market.State, newState market.State) (changed bool, user state.UserData, err error) {
		changes := &marketChanges{}

		dealsPred := pred.OnDealStateChanged(pred.OnDealStateAmtChanged())
		dealsChanged, dealUserData, err := dealsPred(ctx, oldState, newState)
		if err != nil {
			return false, nil, nil
		}
		if dealsChanged {
			dealChanges, ok := dealUserData.(*market.DealStateChanges)
			if !ok {
				// indicates a developer error or breaking change in lotus
				return false, nil, xerrors.Errorf("Unknown type returned by Deal State AMT predicate: %T", dealUserData)
			}
			changes.DealChanges = dealChanges
		}

		proposalsPred := pred.OnDealProposalChanged(pred.OnDealProposalAmtChanged())
		proposalsChanged, proposalsUserData, err := proposalsPred(ctx, oldState, newState)
		if err != nil {
			return false, nil, nil
		}
		if proposalsChanged {
			proposalChanges, ok := proposalsUserData.(*market.DealProposalChanges)
			if !ok {
				// indicates a developer error or breaking change in lotus
				return false, nil, xerrors.Errorf("Unknown type returned by Deal Proposal AMT predicate: %T", dealUserData)
			}
			changes.ProposalChanges = proposalChanges
		}

		if !dealsChanged && !proposalsChanged {
			return false, nil, nil
		}

		return true, state.UserData(changes), nil
	}
}
