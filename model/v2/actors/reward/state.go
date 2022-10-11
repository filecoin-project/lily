package reward

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("reward")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&ChainReward{}, ExtractChainReward)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range reward.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&ChainReward{}, supportedActors)

}

var _ v2.LilyModel = (*ChainReward)(nil)

type ChainReward struct {
	Height                                  abi.ChainEpoch
	StateRoot                               cid.Cid
	ThisEpochBaselinePower                  abi.StoragePower
	ThisEpochReward                         abi.StoragePower
	ThisEpochRewardSmoothedPositionEstimate abi.StoragePower
	ThisEpochRewardSmoothedVelocityEstimate abi.StoragePower
	EffectiveBaselinePower                  abi.StoragePower
	EffectiveNetworkTime                    abi.ChainEpoch
	TotalStoragePowerReward                 abi.TokenAmount
	CumSumBaseline                          abi.StoragePower
	CumSumRealized                          abi.StoragePower
}

func (m *ChainReward) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ChainReward{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *ChainReward) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *ChainReward) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *ChainReward) ToStorageBlock() (block.Block, error) {
	data, err := m.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (m *ChainReward) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractChainReward(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("extractor", "RewardExtractor"), zap.Inline(a))

	rstate, err := reward.Load(api.Store(), &a.Actor)
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

	return []v2.LilyModel{
		&ChainReward{
			Height:                                  current.Height(),
			StateRoot:                               current.ParentState(),
			ThisEpochBaselinePower:                  thisBaslinePower,
			ThisEpochReward:                         thisReward,
			ThisEpochRewardSmoothedPositionEstimate: thisRewardSmoothed.PositionEstimate,
			ThisEpochRewardSmoothedVelocityEstimate: thisRewardSmoothed.VelocityEstimate,
			EffectiveBaselinePower:                  effectiveBaselinePower,
			EffectiveNetworkTime:                    networkTime,
			TotalStoragePowerReward:                 totalMinedReward,
			CumSumBaseline:                          csbaseline,
			CumSumRealized:                          csrealized,
		},
	}, nil
}
