package actorstate

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	commonmodel "github.com/filecoin-project/sentinel-visor/model/actors/common"
)

// was services/processor/tasks/common/actor.go

// ActorExtractor extracts common actor state
type ActorExtractor struct{}

func (ActorExtractor) Extract(ctx context.Context, a ActorInfo, emsgs []*lens.ExecutedMessage, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "ActorExtractor")
	defer span.End()

	ast, err := node.StateReadState(ctx, a.Address, a.TipSet.Key())
	if err != nil {
		return nil, err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil, err
	}

	return &commonmodel.ActorTaskResult{
		Actor: &commonmodel.Actor{
			Height:    int64(a.Epoch),
			ID:        a.Address.String(),
			StateRoot: a.ParentStateRoot.String(),
			Code:      builtin.ActorNameByCode(a.Actor.Code),
			Head:      a.Actor.Head.String(),
			Balance:   a.Actor.Balance.String(),
			Nonce:     a.Actor.Nonce,
		},
		State: &commonmodel.ActorState{
			Height: int64(a.Epoch),
			Head:   a.Actor.Head.String(),
			Code:   a.Actor.Code.String(),
			State:  string(state),
		},
	}, nil
}
