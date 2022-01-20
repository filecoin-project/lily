package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	"github.com/ipfs/go-cid"
)

var rewardAllowed map[cid.Cid]bool

func init() {
	rewardAllowed = make(map[cid.Cid]bool)
	for _, c := range reward.AllCodes() {
		rewardAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&rewardmodel.ChainReward{}, ChainRewardExtractor{})
}

var _ model.ActorStateExtractor = (*ChainRewardExtractor)(nil)

type ChainRewardExtractor struct{}

func (ChainRewardExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	return RewardExtractor{}.Extract(ctx, ActorInfo(actor), api)
}

func (ChainRewardExtractor) Allow(code cid.Cid) bool {
	return rewardAllowed[code]
}

func (ChainRewardExtractor) Name() string {
	return "chain_rewards"
}
