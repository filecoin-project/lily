package v0

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	reward0 "github.com/filecoin-project/specs-actors/actors/builtin/reward"

	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
)

func RewardHandler(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *rawdiff.ActorChange) (model.Persistable, error) {
	if change.Change == core.ChangeTypeRemove {
		panic("reward is a singleton actor and cannot be removed")
	}
	state := new(reward0.State)
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
		TotalMinedReward:                  state.TotalMined.String(),
		NewReward:                         state.ThisEpochReward.String(),
		EffectiveNetworkTime:              int64(state.EffectiveNetworkTime),
	}, nil

}
