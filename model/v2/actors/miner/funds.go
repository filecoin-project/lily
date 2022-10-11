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
	v2.RegisterActorExtractor(&LockedFunds{}, ExtractLockedFunds)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&LockedFunds{}, supportedActors)

}

var _ v2.LilyModel = (*LockedFunds)(nil)

type LockedFunds struct {
	Height                   abi.ChainEpoch
	StateRoot                cid.Cid
	Miner                    address.Address
	VestingFunds             abi.TokenAmount
	InitialPledgeRequirement abi.TokenAmount
	PreCommitDeposits        abi.TokenAmount
}

func (m *LockedFunds) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(LockedFunds{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (m *LockedFunds) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    m.Height,
		StateRoot: m.StateRoot,
	}
}

func (m *LockedFunds) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := m.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *LockedFunds) ToStorageBlock() (block.Block, error) {
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

func (m *LockedFunds) Cid() cid.Cid {
	sb, err := m.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func ExtractLockedFunds(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("model", "LockedFunds"), zap.Inline(a))

	ec, err := miner2.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	currLocked, err := ec.CurrState.LockedFunds()
	if err != nil {
		return nil, fmt.Errorf("loading current miner locked funds: %w", err)
	}
	if ec.HasPreviousState() {
		prevLocked, err := ec.PrevState.LockedFunds()
		if err != nil {
			return nil, fmt.Errorf("loading previous miner locked funds: %w", err)
		}

		// if all values are equal there is no change.
		if prevLocked.VestingFunds.Equals(currLocked.VestingFunds) &&
			prevLocked.PreCommitDeposits.Equals(currLocked.PreCommitDeposits) &&
			prevLocked.InitialPledgeRequirement.Equals(currLocked.InitialPledgeRequirement) {
			return nil, nil
		}
	}

	return []v2.LilyModel{
		&LockedFunds{
			Height:                   current.Height(),
			StateRoot:                current.ParentState(),
			Miner:                    a.Address,
			VestingFunds:             currLocked.VestingFunds,
			InitialPledgeRequirement: currLocked.InitialPledgeRequirement,
			PreCommitDeposits:        currLocked.PreCommitDeposits,
		},
	}, nil
}
