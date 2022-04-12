package actorstate

import (
	"context"
	"unicode/utf8"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"golang.org/x/text/runes"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/adt"

	market "github.com/filecoin-project/lily/chain/actors/builtin/market"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
)

// was services/processor/tasks/market/market.go

// StorageMarketExtractor extracts market actor state
type StorageMarketExtractor struct{}

func init() {
	for _, c := range market.AllCodes() {
		Register(c, StorageMarketExtractor{})
	}
}

type MarketStateExtractionContext struct {
	PrevState market.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState market.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func NewMarketStateExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*MarketStateExtractionContext, error) {
	curState, err := market.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current market state: %w", err)
	}

	prevTipset := a.TipSet
	prevState := curState
	if a.Epoch != 0 {
		prevTipset = a.ParentTipSet

		prevActor, err := node.Actor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state at tipset %s epoch %d: %w", a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = market.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state: %w", err)
		}

	}
	return &MarketStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.TipSet,
		Store:     node.Store(),
	}, nil
}

func (m *MarketStateExtractionContext) IsGenesis() bool {
	return m.CurrTs.Height() == 0
}

func (m StorageMarketExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StorageMarketExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	ec, err := NewMarketStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	dealStateModel, err := ExtractMarketDealStates(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting market deal state changes: %w", err)
	}

	dealProposalModel, err := ExtractMarketDealProposals(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting market proposal changes: %w", err)
	}

	return &marketmodel.MarketTaskResult{
		Proposals: dealProposalModel,
		States:    dealStateModel,
	}, nil
}

func ExtractMarketDealProposals(ctx context.Context, ec *MarketStateExtractionContext) (marketmodel.MarketDealProposals, error) {
	currDealProposals, err := ec.CurrState.Proposals()
	if err != nil {
		return nil, xerrors.Errorf("loading current market deal proposals: %w:", err)
	}

	if ec.IsGenesis() {
		var out marketmodel.MarketDealProposals
		if err := currDealProposals.ForEach(func(id abi.DealID, dp market.DealProposal) error {
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
				Label:                SanitizeLabel(dp.Label),
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking current deal states: %w", err)
		}
		return out, nil

	}

	changed, err := ec.CurrState.ProposalsChanged(ec.PrevState)
	if err != nil {
		return nil, xerrors.Errorf("checking for deal proposal changes: %w", err)
	}

	if !changed {
		return nil, nil
	}

	changes, err := market.DiffDealProposals(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, xerrors.Errorf("diffing deal states: %w", err)
	}

	out := make(marketmodel.MarketDealProposals, len(changes.Added))
	for idx, add := range changes.Added {
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
			Label:                SanitizeLabel(add.Proposal.Label),
		}
	}
	return out, nil
}

func ExtractMarketDealStates(ctx context.Context, ec *MarketStateExtractionContext) (marketmodel.MarketDealStates, error) {
	currDealStates, err := ec.CurrState.States()
	if err != nil {
		return nil, xerrors.Errorf("loading current market deal states: %w", err)
	}

	if ec.IsGenesis() {
		var out marketmodel.MarketDealStates
		if err := currDealStates.ForEach(func(id abi.DealID, ds market.DealState) error {
			out = append(out, &marketmodel.MarketDealState{
				Height:           int64(ec.CurrTs.Height()),
				DealID:           uint64(id),
				SectorStartEpoch: int64(ds.SectorStartEpoch),
				LastUpdateEpoch:  int64(ds.LastUpdatedEpoch),
				SlashEpoch:       int64(ds.SlashEpoch),
				StateRoot:        ec.CurrTs.ParentState().String(),
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking current deal states: %w", err)
		}
		return out, nil
	}

	changed, err := ec.CurrState.StatesChanged(ec.PrevState)
	if err != nil {
		return nil, xerrors.Errorf("checking for deal state changes: %w", err)
	}

	if !changed {
		return nil, nil
	}

	changes, err := market.DiffDealStates(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, xerrors.Errorf("diffing deal states: %w", err)
	}

	out := make(marketmodel.MarketDealStates, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealState{
			Height:           int64(ec.CurrTs.Height()),
			DealID:           uint64(add.ID),
			SectorStartEpoch: int64(add.Deal.SectorStartEpoch),
			LastUpdateEpoch:  int64(add.Deal.LastUpdatedEpoch),
			SlashEpoch:       int64(add.Deal.SlashEpoch),
			StateRoot:        ec.CurrTs.ParentState().String(),
		}
		idx++
	}
	for _, mod := range changes.Modified {
		out[idx] = &marketmodel.MarketDealState{
			Height:           int64(ec.CurrTs.Height()),
			DealID:           uint64(mod.ID),
			SectorStartEpoch: int64(mod.To.SectorStartEpoch),
			LastUpdateEpoch:  int64(mod.To.LastUpdatedEpoch),
			SlashEpoch:       int64(mod.To.SlashEpoch),
			StateRoot:        ec.CurrTs.ParentState().String(),
		}
		idx++
	}
	return out, nil
}

// SanitizeLabel ensures that s is a valid utf8 string by replacing any ill formed bytes with a replacement character.
func SanitizeLabel(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	tr := runes.ReplaceIllFormed()
	return tr.String(s)
}
