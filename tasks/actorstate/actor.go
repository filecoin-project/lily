package actorstate

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	commonmodel "github.com/filecoin-project/sentinel-visor/model/actors/common"
)

// was services/processor/tasks/common/actor.go

// ActorExtracter extracts common actor state
type ActorExtracter struct{}

func (ActorExtracter) Extract(ctx context.Context, a ActorInfo, node lens.API) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "ActorExtracter")
	defer span.End()

	ast, err := node.StateReadState(ctx, a.Address, a.TipSet)
	if err != nil {
		return nil, err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil, err
	}
	log.Debugw("read full actor state", "addr", a.Address.String(), "size", len(state), "code", builtin.ActorNameByCode(a.Actor.Code))

	return &commonmodel.ActorTaskResult{
		Actor: &commonmodel.Actor{
			ID:        a.Address.String(),
			StateRoot: a.ParentStateRoot.String(),
			Code:      builtin.ActorNameByCode(a.Actor.Code),
			Head:      a.Actor.Head.String(),
			Balance:   a.Actor.Balance.String(),
			Nonce:     a.Actor.Nonce,
		},
		State: &commonmodel.ActorState{
			Head:  a.Actor.Head.String(),
			Code:  a.Actor.Code.String(),
			State: string(state),
		},
	}, nil
}
