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
	v2.RegisterActorExtractor(&FeeDebt{}, ExtractDebt)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&FeeDebt{}, supportedActors)

}

var _ v2.LilyModel = (*FeeDebt)(nil)

type FeeDebt struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Miner     address.Address
	Debt      abi.TokenAmount
}

func (m *FeeDebt) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(FeeDebt{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *FeeDebt) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *FeeDebt) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *FeeDebt) ToStorageBlock() (block.Block, error) {
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

func (m *FeeDebt) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractDebt(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "FeeDebt"), zap.Inline(a))
	ec, err := miner2.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	currDebt, err := ec.CurrState.FeeDebt()
	if err != nil {
		return nil, fmt.Errorf("loading current miner fee debt: %w", err)
	}

	if ec.HasPreviousState() {
		prevDebt, err := ec.PrevState.FeeDebt()
		if err != nil {
			return nil, fmt.Errorf("loading previous miner fee debt: %w", err)
		}
		if prevDebt.Equals(currDebt) {
			return nil, nil
		}
	}
	// debt changed

	return []v2.LilyModel{
		&FeeDebt{
			Height:    current.Height(),
			StateRoot: current.ParentState(),
			Miner:     a.Address,
			Debt:      currDebt,
		},
	}, nil
}
