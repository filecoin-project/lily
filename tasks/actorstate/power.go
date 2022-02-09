package actorstate

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
)

// was services/processor/tasks/power/power.go

// StoragePowerExtractor extracts power actor state
type StoragePowerExtractor struct{}

func init() {
	for _, c := range power.AllCodes() {
		Register(c, StoragePowerExtractor{})
	}
}

func NewPowerStateExtractionContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*PowerStateExtractionContext, error) {
	curState, err := power.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current power state: %w", err)
	}

	prevState := curState
	if a.Epoch != 1 {
		prevActor, err := node.Actor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &PowerStateExtractionContext{
					PrevState: prevState,
					CurrState: curState,
					CurrTs:    a.TipSet,
					Store:     node.Store(),
				}, nil
			}
			return nil, xerrors.Errorf("loading previous power actor at tipset %s epoch %d: %w", a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = power.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous power actor state: %w", err)
		}
	}
	return &PowerStateExtractionContext{
		PrevState: prevState,
		CurrState: curState,
		CurrTs:    a.TipSet,
		Store:     node.Store(),
	}, nil
}

type PowerStateExtractionContext struct {
	PrevState power.State
	CurrState power.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func (p *PowerStateExtractionContext) HasPreviousState() bool {
	return !(p.CurrTs.Height() == 1 || p.PrevState == p.CurrState)
}

func (StoragePowerExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StoragePowerExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	ec, err := NewPowerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	chainPowerModel, err := ExtractChainPower(ec)
	if err != nil {
		return nil, err
	}

	claimedPowerModel, err := ExtractClaimedPower(ctx, ec)
	if err != nil {
		return nil, err
	}
	return &powermodel.PowerTaskResult{
		ChainPowerModel: chainPowerModel,
		ClaimStateModel: claimedPowerModel,
	}, nil
}

func ExtractChainPower(ec *PowerStateExtractionContext) (*powermodel.ChainPower, error) {
	locked, err := ec.CurrState.TotalLocked()
	if err != nil {
		return nil, err
	}
	pow, err := ec.CurrState.TotalPower()
	if err != nil {
		return nil, err
	}
	commit, err := ec.CurrState.TotalCommitted()
	if err != nil {
		return nil, err
	}
	smoothed, err := ec.CurrState.TotalPowerSmoothed()
	if err != nil {
		return nil, err
	}
	participating, total, err := ec.CurrState.MinerCounts()
	if err != nil {
		return nil, err
	}

	return &powermodel.ChainPower{
		Height:                     int64(ec.CurrTs.Height()),
		StateRoot:                  ec.CurrTs.ParentState().String(),
		TotalRawBytesPower:         pow.RawBytePower.String(),
		TotalQABytesPower:          pow.QualityAdjPower.String(),
		TotalRawBytesCommitted:     commit.RawBytePower.String(),
		TotalQABytesCommitted:      commit.QualityAdjPower.String(),
		TotalPledgeCollateral:      locked.String(),
		QASmoothedPositionEstimate: smoothed.PositionEstimate.String(),
		QASmoothedVelocityEstimate: smoothed.VelocityEstimate.String(),
		MinerCount:                 total,
		ParticipatingMinerCount:    participating,
	}, nil
}

func ExtractClaimedPower(ctx context.Context, ec *PowerStateExtractionContext) (powermodel.PowerActorClaimList, error) {
	claimModel := powermodel.PowerActorClaimList{}
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClaim(func(miner address.Address, claim power.Claim) error {
			claimModel = append(claimModel, &powermodel.PowerActorClaim{
				Height:          int64(ec.CurrTs.Height()),
				StateRoot:       ec.CurrTs.ParentState().String(),
				MinerID:         miner.String(),
				RawBytePower:    claim.RawBytePower.String(),
				QualityAdjPower: claim.QualityAdjPower.String(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return claimModel, nil
	}
	// normal case.
	claimChanges, err := power.DiffClaims(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	for _, newClaim := range claimChanges.Added {
		claimModel = append(claimModel, &powermodel.PowerActorClaim{
			Height:          int64(ec.CurrTs.Height()),
			StateRoot:       ec.CurrTs.ParentState().String(),
			MinerID:         newClaim.Miner.String(),
			RawBytePower:    newClaim.Claim.RawBytePower.String(),
			QualityAdjPower: newClaim.Claim.QualityAdjPower.String(),
		})
	}
	for _, modClaim := range claimChanges.Modified {
		claimModel = append(claimModel, &powermodel.PowerActorClaim{
			Height:          int64(ec.CurrTs.Height()),
			StateRoot:       ec.CurrTs.ParentState().String(),
			MinerID:         modClaim.Miner.String(),
			RawBytePower:    modClaim.To.RawBytePower.String(),
			QualityAdjPower: modClaim.To.QualityAdjPower.String(),
		})
	}
	return claimModel, nil
}
