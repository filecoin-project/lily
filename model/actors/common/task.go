package common

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type ActorTaskResult struct {
	Actor *Actor
	State *ActorState
}

func (a *ActorTaskResult) Persist(ctx context.Context, s model.StorageBatch) error {
	if err := a.Actor.Persist(ctx, s); err != nil {
		return err
	}
	if err := a.State.Persist(ctx, s); err != nil {
		return err
	}
	return nil
}
