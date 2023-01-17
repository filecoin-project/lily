// Code generated by: `make actors-gen`. DO NOT EDIT.
package reward

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"

	miner10 "github.com/filecoin-project/go-state-types/builtin/v10/miner"
	reward10 "github.com/filecoin-project/go-state-types/builtin/v10/reward"
	smoothing10 "github.com/filecoin-project/go-state-types/builtin/v10/util/smoothing"
)

var _ State = (*state10)(nil)

func load10(store adt.Store, root cid.Cid) (State, error) {
	out := state10{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state10 struct {
	reward10.State
	store adt.Store
}

func (s *state10) ThisEpochReward() (abi.TokenAmount, error) {
	return s.State.ThisEpochReward, nil
}

func (s *state10) ThisEpochRewardSmoothed() (builtin.FilterEstimate, error) {

	return builtin.FilterEstimate{
		PositionEstimate: s.State.ThisEpochRewardSmoothed.PositionEstimate,
		VelocityEstimate: s.State.ThisEpochRewardSmoothed.VelocityEstimate,
	}, nil

}

func (s *state10) ThisEpochBaselinePower() (abi.StoragePower, error) {
	return s.State.ThisEpochBaselinePower, nil
}

func (s *state10) TotalStoragePowerReward() (abi.TokenAmount, error) {
	return s.State.TotalStoragePowerReward, nil
}

func (s *state10) EffectiveBaselinePower() (abi.StoragePower, error) {
	return s.State.EffectiveBaselinePower, nil
}

func (s *state10) EffectiveNetworkTime() (abi.ChainEpoch, error) {
	return s.State.EffectiveNetworkTime, nil
}

func (s *state10) CumsumBaseline() (reward10.Spacetime, error) {
	return s.State.CumsumBaseline, nil
}

func (s *state10) CumsumRealized() (reward10.Spacetime, error) {
	return s.State.CumsumRealized, nil
}

func (s *state10) InitialPledgeForPower(qaPower abi.StoragePower, networkTotalPledge abi.TokenAmount, networkQAPower *builtin.FilterEstimate, circSupply abi.TokenAmount) (abi.TokenAmount, error) {
	return miner10.InitialPledgeForPower(
		qaPower,
		s.State.ThisEpochBaselinePower,
		s.State.ThisEpochRewardSmoothed,
		smoothing10.FilterEstimate{
			PositionEstimate: networkQAPower.PositionEstimate,
			VelocityEstimate: networkQAPower.VelocityEstimate,
		},
		circSupply,
	), nil
}

func (s *state10) PreCommitDepositForPower(networkQAPower builtin.FilterEstimate, sectorWeight abi.StoragePower) (abi.TokenAmount, error) {
	return miner10.PreCommitDepositForPower(s.State.ThisEpochRewardSmoothed,
		smoothing10.FilterEstimate{
			PositionEstimate: networkQAPower.PositionEstimate,
			VelocityEstimate: networkQAPower.VelocityEstimate,
		},
		sectorWeight), nil
}

func (s *state10) ActorKey() string {
	return actors.RewardKey
}

func (s *state10) ActorVersion() actors.Version {
	return actors.Version10
}

func (s *state10) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}
