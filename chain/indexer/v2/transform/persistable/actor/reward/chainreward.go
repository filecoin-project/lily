package reward

import (
	"context"
	"fmt"
	"reflect"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/reward"
)

var log = logging.Logger("transform/reward")

type ChainRewardTransform struct {
	meta v2.ModelMeta
}

func NewChainRewardTransform() *ChainRewardTransform {
	info := reward.ChainReward{}
	return &ChainRewardTransform{meta: info.Meta()}
}

func (s *ChainRewardTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(model.PersistableList, 0, len(res.Models()))
			for _, modeldata := range res.Models() {
				cr := modeldata.(*reward.ChainReward)
				sqlModels = append(sqlModels, &rewardmodel.ChainReward{
					Height:                            int64(cr.Height),
					StateRoot:                         cr.StateRoot.String(),
					CumSumBaseline:                    cr.CumSumBaseline.String(),
					CumSumRealized:                    cr.CumSumRealized.String(),
					EffectiveBaselinePower:            cr.EffectiveBaselinePower.String(),
					NewBaselinePower:                  cr.ThisEpochBaselinePower.String(),
					NewRewardSmoothedPositionEstimate: cr.ThisEpochRewardSmoothedPositionEstimate.String(),
					NewRewardSmoothedVelocityEstimate: cr.ThisEpochRewardSmoothedVelocityEstimate.String(),
					TotalMinedReward:                  cr.TotalStoragePowerReward.String(),
					NewReward:                         cr.ThisEpochReward.String(),
					EffectiveNetworkTime:              int64(cr.EffectiveNetworkTime),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *ChainRewardTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *ChainRewardTransform) Name() string {
	info := ChainRewardTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *ChainRewardTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}
