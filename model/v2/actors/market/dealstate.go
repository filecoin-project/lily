package market

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	marketex "github.com/filecoin-project/lily/tasks/actorstate/market"
)

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&DealState{}, ExtractDealState)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range market.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&DealState{}, supportedActors)
}

type DealState struct {
	Height           abi.ChainEpoch
	StateRoot        cid.Cid
	DealID           abi.DealID
	SectorStartEpoch abi.ChainEpoch
	LastUpdateEpoch  abi.ChainEpoch
	SlashEpoch       abi.ChainEpoch
}

func (t *DealState) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *DealState) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *DealState) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func (t *DealState) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(DealState{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (t *DealState) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func ExtractDealState(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	ec, err := marketex.NewMarketStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, err
	}

	currDealStates, err := ec.CurrState.States()
	if err != nil {
		return nil, fmt.Errorf("loading current market deal states: %w", err)
	}

	if ec.IsGenesis() {
		var out []v2.LilyModel
		if err := currDealStates.ForEach(func(id abi.DealID, ds market.DealState) error {
			out = append(out, &DealState{
				Height:           current.Height(),
				StateRoot:        current.ParentState(),
				DealID:           id,
				SectorStartEpoch: ds.SectorStartEpoch,
				LastUpdateEpoch:  ds.LastUpdatedEpoch,
				SlashEpoch:       ds.SlashEpoch,
			})
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walking current deal states: %w", err)
		}
		return out, nil
	}

	changed, err := ec.CurrState.StatesChanged(ec.PrevState)
	if err != nil {
		return nil, fmt.Errorf("checking for deal state changes: %w", err)
	}

	if !changed {
		return nil, nil
	}

	changes, err := market.DiffDealStates(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing deal states: %w", err)
	}

	out := make([]v2.LilyModel, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &DealState{
			Height:           current.Height(),
			StateRoot:        current.ParentState(),
			DealID:           add.ID,
			SectorStartEpoch: add.Deal.SectorStartEpoch,
			LastUpdateEpoch:  add.Deal.LastUpdatedEpoch,
			SlashEpoch:       add.Deal.SlashEpoch,
		}
		idx++
	}
	for _, mod := range changes.Modified {
		out[idx] = &DealState{
			Height:           ec.CurrTs.Height(),
			StateRoot:        ec.CurrTs.ParentState(),
			DealID:           mod.ID,
			SectorStartEpoch: mod.To.SectorStartEpoch,
			LastUpdateEpoch:  mod.To.LastUpdatedEpoch,
			SlashEpoch:       mod.To.SlashEpoch,
		}
		idx++
	}
	return out, nil
}
