package chain

import (
	"context"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func StringToModelTypeAndExtractor(name string) (model.Persistable, ExtractorType, error) {
	switch name {
	case "block_headers":
		return &blocks.BlockHeader{}, TipSetStateExtractorType, nil
	case "block_parents":
		return &blocks.BlockParent{}, TipSetStateExtractorType, nil
	case "drand_block_entries":
		return &blocks.DrandBlockEntrie{}, TipSetStateExtractorType, nil
	case "actors":
		return &common.Actor{}, ActorStateExtractorType, nil
	case "actor_states":
		return &common.ActorState{}, ActorStateExtractorType, nil
	default:
		return nil, UnknownStateExtractorType, xerrors.Errorf("unknown model name %s", name)
	}
}

type ExtractorType string

var UnknownStateExtractorType ExtractorType = "Unknown"
var TipSetStateExtractorType ExtractorType = "TipSetStateExtractor"
var ActorStateExtractorType ExtractorType = "ActorStateExtractor"

type ExtractableModel interface {
	Type() ExtractorType
}

type TipSetStateExtractorFactory interface {
	NewExtractor() TipSetStateExtractor
}

type TipSetStateExtractor interface {
	Extract(ctx context.Context, current, previous *types.TipSet) (model.Persistable, error)
}

type ActorStateExtractorFactory interface {
	NewExtractor() ActorStateExtractor
}

type ActorStateExtractor interface {
	Extract(ctx context.Context, actor actorstate.ActorInfo, api actorstate.ActorStateAPI) (model.Persistable, error)
	Allow(code cid.Cid) bool
	Name() string
}

func ActorStateExtractorForModel(m model.Persistable) ActorStateExtractor {
	switch m.(type) {
	case *common.Actor:
		return &ActorExtractor{}
	default:
		panic("here")
	}
}

func TipSetExtractorForModel(m model.Persistable) TipSetStateExtractor {
	return nil
}

type ActorExtractor struct{}

func (ae *ActorExtractor) Extract(ctx context.Context, act actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	a := new(common.Actor)
	a.Height = int64(act.TipSet.Height())
	a.ID = act.Address.String()
	a.StateRoot = act.ParentStateRoot.String()
	a.Code = act.Actor.Code.String()
	a.Head = act.Actor.Head.String()
	a.Balance = act.Actor.Balance.String()
	a.Nonce = act.Actor.Nonce
	return a, nil
}

func (aw *ActorExtractor) Allow(code cid.Cid) bool {
	// Allow all actor types
	return true
}

func (aw *ActorExtractor) Name() string {
	return "actors"
}
