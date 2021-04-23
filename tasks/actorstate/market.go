package actorstate

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-amt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	market "github.com/filecoin-project/lotus/chain/actors/builtin/market"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa0market "github.com/filecoin-project/specs-actors/actors/builtin/market"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa2market "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	sa3market "github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	sa4market "github.com/filecoin-project/specs-actors/v4/actors/builtin/market"

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
	Register(sa3builtin.StorageMarketActorCodeID, StorageMarketExtractor{})
	Register(sa4builtin.StorageMarketActorCodeID, StorageMarketExtractor{})
}

type MarketStateExtractionContext struct {
	PrevActor *types.Actor
	PrevState market.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState market.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func NewMarketStateExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*MarketStateExtractionContext, error) {
	curActor, err := node.StateGetActor(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current market actor: %w", err)
	}

	curTipset, err := node.ChainGetTipSet(ctx, a.TipSet)
	if err != nil {
		return nil, xerrors.Errorf("loading current tipset: %w", err)
	}

	curState, err := market.Load(node.Store(), curActor)
	if err != nil {
		return nil, xerrors.Errorf("loading current market state: %w", err)
	}

	prevTipset := curTipset
	prevState := curState
	prevActor := curActor
	if a.Epoch != 0 {
		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet)
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state at tipset %s epoch %d: %w", a.ParentTipSet, a.Epoch, err)
		}

		prevState, err = market.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state: %w", err)
		}

		prevTipset, err = node.ChainGetTipSet(ctx, a.ParentTipSet)
		if err != nil {
			return nil, xerrors.Errorf("loading previous tipset: %w", err)
		}
	}
	return &MarketStateExtractionContext{
		PrevActor: prevActor,
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: curActor,
		CurrState: curState,
		CurrTs:    curTipset,
		Store:     node.Store(),
	}, nil
}

func (m *MarketStateExtractionContext) IsGenesis() bool {
	return m.CurrTs.Height() == 0
}

func (m StorageMarketExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMarketExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
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
				Label:                dp.Label,
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

	return DiffDealProposals(ctx, ec)

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

	return DiffDealStates(ctx, ec)
}

func DiffDealStates(ctx context.Context, ec *MarketStateExtractionContext) (marketmodel.MarketDealStates, error) {
	prevS, err := ec.PrevState.States()
	if err != nil {
		return nil, err
	}
	prevRoot, err := prevS.Root()
	if err != nil {
		return nil, err
	}
	prevBitwidth, err := dealStateAMTBitwidth(ec.PrevActor)
	if err != nil {
		return nil, xerrors.Errorf("bitwidth previous state: %w", err)
	}
	currS, err := ec.CurrState.States()
	if err != nil {
		return nil, err
	}
	currRoot, err := currS.Root()
	if err != nil {
		return nil, err
	}
	currBitwidth, err := dealStateAMTBitwidth(ec.CurrActor)
	if err != nil {
		return nil, xerrors.Errorf("bitwidth current state: %w", err)
	}

	if prevBitwidth != currBitwidth {
		log.Errorw("cannot diff deal state amt's with different bitwidths", "prevTS", ec.PrevTs, "currTS", ec.CurrTs)
		return nil, nil
	}

	prevAMT, err := amt.LoadAMT(ctx, ec.Store, prevRoot, amt.UseTreeBitWidth(prevBitwidth))
	if err != nil {
		return nil, err
	}

	currAMT, err := amt.LoadAMT(ctx, ec.Store, currRoot, amt.UseTreeBitWidth(currBitwidth))
	if err != nil {
		return nil, err
	}

	// TODO this will fail if on prevRoot uses a different BitWidth than currRoot
	changes, err := amt.DiffAMT(ctx, ec.Store, ec.Store, prevAMT, currAMT)
	if err != nil {
		return nil, err
	}

	out := make(marketmodel.MarketDealStates, 0, len(changes))
	for _, change := range changes {
		// only interested in states that have been added or modified.
		if change.Type == amt.Remove {
			continue
		}
		ds, err := unmarshalDealStates(change.After, ec.CurrActor)
		if err != nil {
			return nil, err
		}
		out = append(out,
			&marketmodel.MarketDealState{
				Height:           int64(ec.CurrTs.Height()),
				DealID:           change.Key,
				SectorStartEpoch: int64(ds.SectorStartEpoch),
				LastUpdateEpoch:  int64(ds.LastUpdatedEpoch),
				SlashEpoch:       int64(ds.SlashEpoch),
				StateRoot:        ec.CurrTs.ParentState().String(),
			},
		)
	}
	return out, nil

}

func DiffDealProposals(ctx context.Context, ec *MarketStateExtractionContext) (marketmodel.MarketDealProposals, error) {
	prevP, err := ec.PrevState.Proposals()
	if err != nil {
		return nil, err
	}
	prevRoot, err := prevP.Root()
	if err != nil {
		return nil, err
	}
	prevBitwidth, err := dealProposalAMTBitWidth(ec.PrevActor)
	if err != nil {
		return nil, err
	}

	currP, err := ec.CurrState.Proposals()
	if err != nil {
		return nil, err
	}
	currRoot, err := currP.Root()
	if err != nil {
		return nil, err
	}
	currBitwidth, err := dealProposalAMTBitWidth(ec.CurrActor)
	if err != nil {
		return nil, err
	}

	if prevBitwidth != currBitwidth {
		log.Errorw("cannot diff deal proposals amt's with different bitwidths", "prevTS", ec.PrevTs, "currTS", ec.CurrTs)
		return nil, nil
	}

	prevAMT, err := amt.LoadAMT(ctx, ec.Store, prevRoot, amt.UseTreeBitWidth(prevBitwidth))
	if err != nil {
		return nil, err
	}

	currAMT, err := amt.LoadAMT(ctx, ec.Store, currRoot, amt.UseTreeBitWidth(currBitwidth))
	if err != nil {
		return nil, err
	}

	// TODO this will fail if on prevRoot uses a different BitWidth than currRoot
	changes, err := amt.DiffAMT(ctx, ec.Store, ec.Store, prevAMT, currAMT)
	if err != nil {
		return nil, err
	}

	out := make(marketmodel.MarketDealProposals, 0, len(changes))
	for _, change := range changes {
		// Deal Proposals are immutable, we only care about new ones
		if change.Type != amt.Add {
			continue
		}
		dp, err := unmarshalDealProposal(change.After, ec.CurrActor)
		if err != nil {
			return nil, err
		}
		out = append(out,
			&marketmodel.MarketDealProposal{
				Height:               int64(ec.CurrTs.Height()),
				DealID:               change.Key,
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
				Label:                dp.Label,
			},
		)
	}
	return out, nil
}

// TODO would be great if there was a source of truth for the bitwidths used at each version *angry noises*
func dealProposalAMTBitWidth(a *types.Actor) (uint, error) {
	switch a.Code {
	case sa0builtin.StorageMarketActorCodeID:
		return 8, nil // wild guess
	case sa2builtin.StorageMarketActorCodeID:
		return 8, nil // lucky number maybe?
	case sa3builtin.StorageMarketActorCodeID:
		return sa3market.ProposalsAmtBitwidth, nil // YAA! * happy sounds *
	case sa4builtin.StorageMarketActorCodeID:
		return sa4market.ProposalsAmtBitwidth, nil
	}
	return 0, xerrors.Errorf("deal proposal bits unknown actor code: %s", a.Code.String())
}

// TODO would be great if there was a source of truth for the bitwidths used at each version *angry noises*
func dealStateAMTBitwidth(a *types.Actor) (uint, error) {
	switch a.Code {
	case sa0builtin.StorageMarketActorCodeID:
		return 8, nil // wild guess
	case sa2builtin.StorageMarketActorCodeID:
		return 8, nil // lucky number maybe?
	case sa3builtin.StorageMarketActorCodeID:
		return sa3market.StatesAmtBitwidth, nil
	case sa4builtin.StorageMarketActorCodeID:
		return sa4market.StatesAmtBitwidth, nil
	}
	return 0, xerrors.Errorf("deal state bits unknown actor code: %s", a.Code.String())
}

func unmarshalDealStates(raw *cbg.Deferred, a *types.Actor) (*market.DealState, error) {
	switch a.Code {
	case sa0builtin.StorageMarketActorCodeID:
		var out sa0market.DealState
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealState)(&out), nil
	case sa2builtin.StorageMarketActorCodeID:
		var out sa2market.DealState
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealState)(&out), nil
	case sa3builtin.StorageMarketActorCodeID:
		var out sa3market.DealState
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealState)(&out), nil
	case sa4builtin.StorageMarketActorCodeID:
		var out sa4market.DealState
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealState)(&out), nil
	}
	return nil, xerrors.Errorf("unmarshal deal state unknown actor code: %s", a.Code.String())

}

func unmarshalDealProposal(raw *cbg.Deferred, a *types.Actor) (*market.DealProposal, error) {
	switch a.Code {
	case sa0builtin.StorageMarketActorCodeID:
		var out sa0market.DealProposal
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealProposal)(&out), nil
	case sa2builtin.StorageMarketActorCodeID:
		var out sa2market.DealProposal
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealProposal)(&out), nil
	case sa3builtin.StorageMarketActorCodeID:
		var out sa3market.DealProposal
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
		return (*market.DealProposal)(&out), nil
	case sa4builtin.StorageMarketActorCodeID:
		var out sa4market.DealProposal
		if err := out.UnmarshalCBOR(bytes.NewReader(raw.Raw)); err != nil {
			return nil, err
		}
	}
	return nil, xerrors.Errorf("unmarshal deal proposal unknown actor code: %s", a.Code.String())
}
