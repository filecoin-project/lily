package actorstate

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	commonmodel "github.com/filecoin-project/sentinel-visor/model/actors/common"
)

// was services/processor/tasks/common/actor.go

// ActorExtractor extracts common actor state
type ActorExtractor struct{}

func (ActorExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.PersistableWithTx, error) {
	ctx, span := global.Tracer("").Start(ctx, "ActorExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ast, err := node.StateReadState(ctx, a.Address, a.TipSet)
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
			Code:      ActorNameByCode(a.Actor.Code),
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
