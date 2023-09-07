package raw

import (
	"context"
	"encoding/json"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/model"
	commonmodel "github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/rawactor")

// RawActorExtractor extracts common actor state
type RawActorExtractor struct{}

func getState(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) []byte {
	if a.ChangeType == tasks.ChangeTypeRemove {
		return nil
	}

	ast, err := node.ActorState(ctx, a.Address, a.Current)
	if err != nil {
		return nil
	}

	state, err := json.Marshal(ast.State)
	if err != nil {
		return nil
	}
	return state
}

func (RawActorExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("Extract", zap.String("extractor", "RawActorExtractor"), zap.Inline(a))

	_, span := otel.Tracer("").Start(ctx, "RawActorExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	state := getState(ctx, a, node)
	stateStr := ""
	if state != nil {
		stateStr = string(state)
	}

	return &commonmodel.Actor{
		Height:    int64(a.Current.Height()),
		ID:        a.Address.String(),
		StateRoot: a.Current.ParentState().String(),
		Code:      builtin.ActorNameByCode(a.Actor.Code),
		Head:      a.Actor.Head.String(),
		Balance:   a.Actor.Balance.String(),
		Nonce:     a.Actor.Nonce,
		State:     stateStr,
		CodeCID:   a.Actor.Code.String(),
	}, nil
}

func (RawActorExtractor) Transform(_ context.Context, data model.PersistableList) (model.PersistableList, error) {
	actorList := make(commonmodel.ActorList, 0, len(data))
	for _, d := range data {
		a, ok := d.(*commonmodel.Actor)
		if !ok {
			return nil, fmt.Errorf("expected Actor type but got: %T", d)
		}
		actorList = append(actorList, a)
	}
	return model.PersistableList{actorList}, nil
}
