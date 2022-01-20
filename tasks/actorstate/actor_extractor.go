package actorstate

import (
	"context"
	"encoding/json"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/common"
	"github.com/ipfs/go-cid"
)

var _ model.ActorStateExtractor = (*CommonActorExtractor)(nil)

type CommonActorExtractor struct{}

func init() {
	model.RegisterActorModelExtractor(&common.Actor{}, CommonActorExtractor{})
	model.RegisterActorModelExtractor(&common.ActorState{}, CommonActorStateExtractor{})
}

func (CommonActorExtractor) Extract(ctx context.Context, act model.ActorInfo, node model.ActorStateAPI) (model.Persistable, error) {
	a := &common.Actor{
		Height:    int64(act.TipSet.Height()),
		ID:        act.Address.String(),
		StateRoot: act.ParentStateRoot.String(),
		Code:      act.Actor.Code.String(),
		Head:      act.Actor.Head.String(),
		Balance:   act.Actor.Balance.String(),
		Nonce:     act.Actor.Nonce,
	}
	return a, nil
}

func (CommonActorExtractor) Allow(code cid.Cid) bool {
	// Allow all actor types
	return true
}

func (CommonActorExtractor) Name() string {
	return "actors"
}

var _ model.ActorStateExtractor = (*CommonActorStateExtractor)(nil)

type CommonActorStateExtractor struct{}

func (CommonActorStateExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	// Don't attempt to read state if the actor has been deleted
	if actor.ChangeType == lens.ChangeTypeRemove {
		return model.NoData, nil
	}

	ast, err := api.StateReadState(ctx, actor.Address, actor.TipSet.Key())
	if err != nil {
		return nil, err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil, err
	}

	return &common.ActorState{
		Height: int64(actor.Epoch),
		Head:   actor.Actor.Head.String(),
		Code:   actor.Actor.Code.String(),
		State:  string(state),
	}, nil

}

func (CommonActorStateExtractor) Allow(code cid.Cid) bool {
	return true
}

func (CommonActorStateExtractor) Name() string {
	return "actor_states"
}
