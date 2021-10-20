package actorstate

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/lily/lens/util"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	commonmodel "github.com/filecoin-project/lily/model/actors/common"
)

// was services/processor/tasks/common/actor.go

// ActorExtractor extracts common actor state
type ActorExtractor struct{}

func (ActorExtractor) Extract(ctx context.Context, a ActorInfo, emsgs []*lens.ExecutedMessage, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "ActorExtractor")
	defer span.End()

	result := &commonmodel.ActorTaskResult{
		Actor: &commonmodel.Actor{
			Height:    int64(a.Epoch),
			ID:        a.Address.String(),
			StateRoot: a.ParentStateRoot.String(),
			Code:      builtin.ActorNameByCode(a.Actor.Code),
			Head:      a.Actor.Head.String(),
			Balance:   a.Actor.Balance.String(),
			Nonce:     a.Actor.Nonce,
		},
	}

	// Don't attempt to read state if the actor has been deleted
	if a.ChangeType == util.ChangeTypeRemove {
		return result, nil
	}

	ast, err := node.StateReadState(ctx, a.Address, a.TipSet.Key())
	if err != nil {
		return nil, err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil, err
	}

	result.State = &commonmodel.ActorState{
		Height: int64(a.Epoch),
		Head:   a.Actor.Head.String(),
		Code:   a.Actor.Code.String(),
		State:  string(state),
	}

	return result, nil
}
