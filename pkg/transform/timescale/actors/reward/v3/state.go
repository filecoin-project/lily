package v3

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	reward3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/reward"

	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
)

func RewardHandler(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange) (model.Persistable, error) {
	if change.Change == core.ChangeTypeRemove {
		panic("reward is a singleton actor and cannot be removed")
	}
	state := new(reward3.State)
	if err := state.UnmarshalCBOR(bytes.NewReader(change.Current)); err != nil {
		return nil, err
	}
	return &rewardmodel.ChainReward{
		Height:                            int64(current.Height()),
		StateRoot:                         current.ParentState().String(),
		CumSumBaseline:                    state.CumsumBaseline.String(),
		CumSumRealized:                    state.CumsumRealized.String(),
		EffectiveBaselinePower:            state.EffectiveBaselinePower.String(),
		NewBaselinePower:                  state.ThisEpochBaselinePower.String(),
		NewRewardSmoothedPositionEstimate: state.ThisEpochRewardSmoothed.PositionEstimate.String(),
		NewRewardSmoothedVelocityEstimate: state.ThisEpochRewardSmoothed.VelocityEstimate.String(),
		TotalMinedReward:                  state.TotalStoragePowerReward.String(),
		NewReward:                         state.ThisEpochReward.String(),
		EffectiveNetworkTime:              int64(state.EffectiveNetworkTime),
	}, nil

}
