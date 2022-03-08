package power

import (
	"context"

	"github.com/filecoin-project/go-address"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var _ actorstate.ActorStateExtractor = (*ClaimedPowerExtractor)(nil)

type ClaimedPowerExtractor struct{}

func (c ClaimedPowerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "ClaimedPowerExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "ClaimedPowerExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	ec, err := NewPowerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}
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
