package common

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type ActorTaskResult struct {
	Actor *Actor
	State *ActorState
}

func (a *ActorTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if err := a.Actor.Persist(ctx, s, version); err != nil {
		return err
	}
	if err := a.State.Persist(ctx, s, version); err != nil {
		return err
	}
	return nil
}
