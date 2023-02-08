package rawdiff

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/actor")

type ActorChange struct {
	Actor    *types.Actor    `cborgen:"actor"`
	Current  []byte          `cborgen:"current_state"`
	Previous []byte          `cborgen:"previous_state"`
	Change   core.ChangeType `cborgen:"change"`
}

const KindActorChange = "actor_change"

func (a *ActorChange) Kind() actors.ActorStateKind {
	return KindActorChange
}

type Actor struct{}

func (Actor) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindActorChange, zap.Inline(act), "duration", time.Since(start))
	}()
	return ActorDiff(ctx, api, act)
}

func (Actor) Type() string {
	return KindActorChange
}

func ActorDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	switch act.Type {
	case core.ChangeTypeAdd:
		currentState, err := api.ChainReadObj(ctx, act.Current.Head)
		if err != nil {
			return nil, err
		}
		return &ActorChange{
			Actor:    act.Current,
			Current:  currentState,
			Previous: nil,
			Change:   act.Type,
		}, nil
	case core.ChangeTypeRemove:
		executedState, err := api.ChainReadObj(ctx, act.Executed.Head)
		if err != nil {
			return nil, err
		}
		return &ActorChange{
			Actor:    act.Executed,
			Current:  nil,
			Previous: executedState,
			Change:   act.Type,
		}, nil
	case core.ChangeTypeModify:
		currentState, err := api.ChainReadObj(ctx, act.Current.Head)
		if err != nil {
			return nil, err
		}
		executedState, err := api.ChainReadObj(ctx, act.Executed.Head)
		if err != nil {
			return nil, err
		}
		return &ActorChange{
			Actor:    act.Current,
			Current:  currentState,
			Previous: executedState,
			Change:   act.Type,
		}, nil
	}
	return nil, fmt.Errorf("unknown actor change type %s", act.Type)
}
