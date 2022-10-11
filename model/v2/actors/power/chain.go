package power

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	power2 "github.com/filecoin-project/lily/tasks/actorstate/power"
)

var log = logging.Logger("power")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&ChainPower{}, ExtractChainPower)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range power.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&ChainPower{}, supportedActors)

}

var _ v2.LilyModel = (*ChainPower)(nil)

type ChainPower struct {
	Height                                  abi.ChainEpoch
	StateRoot                               cid.Cid
	TotalRawBytePower                       abi.StoragePower
	TotalQualityAdjustedBytePower           abi.StoragePower
	TotalRawBytesCommitted                  abi.StoragePower
	TotalQualityAdjustedBytesCommitted      abi.StoragePower
	TotalPledgeCollateral                   abi.TokenAmount
	QualityAdjustedSmoothedPositionEstimate big.Int
	QualityAdjustedSmoothedVelocityEstimate big.Int
	MinerCount                              uint64
	MinerAboveMinPowerCount                 uint64
}

func (m *ChainPower) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ChainPower{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *ChainPower) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *ChainPower) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *ChainPower) ToStorageBlock() (block.Block, error) {
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

func (m *ChainPower) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractChainPower(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "ChainPower"), zap.Inline(a))

	ec, err := power2.NewPowerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, err
	}
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

	return []v2.LilyModel{
		&ChainPower{
			Height:                                  current.Height(),
			StateRoot:                               current.ParentState(),
			TotalRawBytePower:                       pow.RawBytePower,
			TotalQualityAdjustedBytePower:           pow.QualityAdjPower,
			TotalRawBytesCommitted:                  commit.RawBytePower,
			TotalQualityAdjustedBytesCommitted:      commit.QualityAdjPower,
			TotalPledgeCollateral:                   locked,
			QualityAdjustedSmoothedPositionEstimate: smoothed.PositionEstimate,
			QualityAdjustedSmoothedVelocityEstimate: smoothed.VelocityEstimate,
			MinerCount:                              total,
			MinerAboveMinPowerCount:                 participating,
		},
	}, nil
}
