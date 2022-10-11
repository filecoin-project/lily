package miner

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	miner2 "github.com/filecoin-project/lily/tasks/actorstate/miner"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&DeadlineInfo{}, ExtractCurrentDeadline)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&DeadlineInfo{}, supportedActors)

}

var _ v2.LilyModel = (*DeadlineInfo)(nil)

type DeadlineInfo struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Miner     address.Address

	CurrentEpoch abi.ChainEpoch
	PeriodStart  abi.ChainEpoch
	Index        uint64
	Open         abi.ChainEpoch
	Close        abi.ChainEpoch
	Challenge    abi.ChainEpoch
	FaultCutoff  abi.ChainEpoch

	WPoStPeriodDeadlines   uint64
	WPoStProvingPeriod     abi.ChainEpoch
	WPoStChallengeWindow   abi.ChainEpoch
	WPoStChallengeLookback abi.ChainEpoch
	FaultDeclarationCutoff abi.ChainEpoch
}

func (m *DeadlineInfo) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(DeadlineInfo{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *DeadlineInfo) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *DeadlineInfo) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *DeadlineInfo) ToStorageBlock() (block.Block, error) {
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

func (m *DeadlineInfo) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractCurrentDeadline(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "DeadlineInfo"), zap.Inline(a))

	ec, err := miner2.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}
	currDeadlineInfo, err := ec.CurrState.DeadlineInfo(ec.CurrTs.Height())
	if err != nil {
		return nil, err
	}

	if ec.HasPreviousState() {
		prevDeadlineInfo, err := ec.PrevState.DeadlineInfo(ec.CurrTs.Height())
		if err != nil {
			return nil, err
		}
		// TODO implement equality function
		// dereference pointers to check equality
		// if these are different then return a model in the bottom of function
		if prevDeadlineInfo != nil &&
			currDeadlineInfo != nil &&
			*prevDeadlineInfo == *currDeadlineInfo {
			return nil, nil
		}
	}

	// if there is no previous state and the deadlines have changed, return a model
	return []v2.LilyModel{
		&DeadlineInfo{
			Height:                 current.Height(),
			StateRoot:              current.ParentState(),
			Miner:                  a.Address,
			CurrentEpoch:           currDeadlineInfo.CurrentEpoch,
			PeriodStart:            currDeadlineInfo.PeriodStart,
			Index:                  currDeadlineInfo.Index,
			Open:                   currDeadlineInfo.Open,
			Close:                  currDeadlineInfo.Close,
			Challenge:              currDeadlineInfo.Challenge,
			FaultCutoff:            currDeadlineInfo.FaultCutoff,
			WPoStPeriodDeadlines:   currDeadlineInfo.WPoStPeriodDeadlines,
			WPoStProvingPeriod:     currDeadlineInfo.WPoStProvingPeriod,
			WPoStChallengeWindow:   currDeadlineInfo.WPoStChallengeWindow,
			WPoStChallengeLookback: currDeadlineInfo.WPoStChallengeLookback,
			FaultDeclarationCutoff: currDeadlineInfo.FaultDeclarationCutoff,
		},
	}, nil

}
