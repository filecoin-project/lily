package raw

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/model"
	commonmodel "github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type RawActorStateExtractor struct{}

func (RawActorStateExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("Extract", zap.String("extractor", "RawStateActorExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "RawActorStateExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	// Don't attempt to read state if the actor has been deleted
	if a.ChangeType == tasks.ChangeTypeRemove {
		return nil, nil
	}

	ast, err := node.ActorState(ctx, a.Address, a.Current)
	if err != nil {
		return nil, err
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil, err
	}

	return &commonmodel.ActorState{
		Height: int64(a.Current.Height()),
		Head:   a.Actor.Head.String(),
		Code:   a.Actor.Code.String(),
		State:  string(state),
	}, nil
}

func (RawActorStateExtractor) Transform(ctx context.Context, data model.PersistableList) (model.PersistableList, error) {
	actorStateList := make(commonmodel.ActorStateList, 0, len(data))
	for _, d := range data {
		a, ok := d.(*commonmodel.ActorState)
		if !ok {
			return nil, fmt.Errorf("expected Actor type but got: %T", d)
		}
		actorStateList = append(actorStateList, a)
	}
	return model.PersistableList{actorStateList}, nil
}
