package actorstate

import (
	"bytes"
	"context"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/reward"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	rewardmodel "github.com/filecoin-project/sentinel-visor/model/actors/reward"
)

// was services/processor/tasks/reward/reward.go

// RewardExtracter extracts reward actor state
type RewardExtracter struct{}

func init() {
	Register(builtin.RewardActorCodeID, RewardExtracter{})
}

func (RewardExtracter) Extract(ctx context.Context, a ActorInfo, node lens.API) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "RewardExtracter")
	defer span.End()

	rewardStateRaw, err := node.ChainReadObj(ctx, a.Actor.Head)
	if err != nil {
		return nil, err
	}

	var rwdState reward.State
	if err := rwdState.UnmarshalCBOR(bytes.NewReader(rewardStateRaw)); err != nil {
		return nil, err
	}

	return &rewardmodel.ChainReward{
		StateRoot:                         a.ParentStateRoot.String(),
		CumSumBaseline:                    rwdState.CumsumBaseline.String(),
		CumSumRealized:                    rwdState.CumsumRealized.String(),
		EffectiveBaselinePower:            rwdState.EffectiveBaselinePower.String(),
		NewBaselinePower:                  rwdState.ThisEpochBaselinePower.String(),
		NewRewardSmoothedPositionEstimate: rwdState.ThisEpochRewardSmoothed.PositionEstimate.String(),
		NewRewardSmoothedVelocityEstimate: rwdState.ThisEpochRewardSmoothed.VelocityEstimate.String(),
		TotalMinedReward:                  rwdState.TotalMined.String(),
		NewReward:                         rwdState.ThisEpochReward.String(),
		EffectiveNetworkTime:              int64(rwdState.EffectiveNetworkTime),
	}, nil
}
