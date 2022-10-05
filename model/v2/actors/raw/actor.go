package raw

import (
	"bytes"
	"context"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"

	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

func mergeMaps(ms ...map[string]cid.Cid) map[string][]cid.Cid {
	out := make(map[string][]cid.Cid)
	for _, m := range ms {
		for k, v := range m {
			out[k] = append(out[k], v)
		}
	}
	return out
}

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&ActorState{}, Extract)
	act0, err := actors.GetActorCodeIDs(actors.Version0)
	if err != nil {
		panic(err)
	}
	act2, err := actors.GetActorCodeIDs(actors.Version2)
	if err != nil {
		panic(err)
	}
	act3, err := actors.GetActorCodeIDs(actors.Version3)
	if err != nil {
		panic(err)
	}
	act4, err := actors.GetActorCodeIDs(actors.Version4)
	if err != nil {
		panic(err)
	}
	act5, err := actors.GetActorCodeIDs(actors.Version5)
	if err != nil {
		panic(err)
	}
	act6, err := actors.GetActorCodeIDs(actors.Version6)
	if err != nil {
		panic(err)
	}
	act7, err := actors.GetActorCodeIDs(actors.Version7)
	if err != nil {
		panic(err)
	}
	act8, err := actors.GetActorCodeIDs(actors.Version8)
	if err != nil {
		panic(err)
	}
	allActors := mergeMaps(act0, act2, act3, act4, act5, act6, act7, act8)

	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, vs := range allActors {
		for _, v := range vs {
			supportedActors.Add(v)
		}
	}
	v2.RegisterActorType(&ActorState{}, supportedActors)
}

var _ v2.LilyModel = (*ActorState)(nil)

type ActorState struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Address   address.Address
	Head      cid.Cid
	Code      cid.Cid
	Nonce     uint64
	Balance   abi.TokenAmount
	State     []byte
}

func (t *ActorState) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *ActorState) ToStorageBlock() (block.Block, error) {
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

func (t *ActorState) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		panic(err)
	}

	return sb.Cid()
}

func (a *ActorState) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(ActorState{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (a *ActorState) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    a.Height,
		StateRoot: a.StateRoot,
	}
}

func Extract(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	state, err := api.ChainReadObj(ctx, a.Actor.Head)
	if err != nil {
		return nil, err
	}
	return []v2.LilyModel{&ActorState{
		Height:    current.Height(),
		StateRoot: current.ParentState(),
		Address:   a.Address,
		Head:      a.Actor.Head,
		Code:      a.Actor.Code,
		Nonce:     a.Actor.Nonce,
		Balance:   a.Actor.Balance,
		State:     state,
	}}, nil
}
