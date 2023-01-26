package reward

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
)

var log = logging.Logger("lily/tasks/reward")

// RewardExtractor extracts reward actor state
type RewardExtractor struct{}

func (RewardExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "RewardExtractor"), zap.Inline(a))
	_, span := otel.Tracer("").Start(ctx, "RewardExtractor.Transform")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	rstate, err := reward.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, err
	}

	csbaseline, err := rstate.CumsumBaseline()
	if err != nil {
		return nil, err
	}

	csrealized, err := rstate.CumsumRealized()
	if err != nil {
		return nil, err
	}
	effectiveBaselinePower, err := rstate.EffectiveBaselinePower()
	if err != nil {
		return nil, err
	}

	thisBaslinePower, err := rstate.ThisEpochBaselinePower()
	if err != nil {
		return nil, err
	}

	thisRewardSmoothed, err := rstate.ThisEpochRewardSmoothed()
	if err != nil {
		return nil, err
	}

	totalMinedReward, err := rstate.TotalStoragePowerReward()
	if err != nil {
		return nil, err
	}

	thisReward, err := rstate.ThisEpochReward()
	if err != nil {
		return nil, err
	}

	networkTime, err := rstate.EffectiveNetworkTime()
	if err != nil {
		return nil, err
	}

	return &rewardmodel.ChainReward{
		Height:                            int64(a.Current.Height()),
		StateRoot:                         a.Current.ParentState().String(),
		CumSumBaseline:                    csbaseline.String(),
		CumSumRealized:                    csrealized.String(),
		EffectiveBaselinePower:            effectiveBaselinePower.String(),
		NewBaselinePower:                  thisBaslinePower.String(),
		NewRewardSmoothedPositionEstimate: thisRewardSmoothed.PositionEstimate.String(),
		NewRewardSmoothedVelocityEstimate: thisRewardSmoothed.VelocityEstimate.String(),
		TotalMinedReward:                  totalMinedReward.String(),
		NewReward:                         thisReward.String(),
		EffectiveNetworkTime:              int64(networkTime),
	}, nil
}
