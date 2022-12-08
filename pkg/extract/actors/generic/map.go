package generic

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type ActorStateLoader = func(adt.Store, *types.Actor) (interface{}, error)
type ActorStateMapLoader = func(interface{}) (adt.Map, *adt.MapOpts, error)

type MapChange struct {
	Key    string
	Value  typegen.Deferred
	Change core.ChangeType
}

type MapChangeList = []*MapChange

func DiffActorMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, actorStateLoader ActorStateLoader, actorMapLoader ActorStateMapLoader) (*core.MapDiff, error) {
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}

	currentActor, err := actorStateLoader(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}

	currentMap, currentMapOpts, err := actorMapLoader(currentActor)
	if err != nil {
		return nil, err
	}
	if act.Type == core.ChangeTypeAdd {
		out := &core.MapDiff{
			Added:    make([]*core.MapChange, 0),
			Modified: make([]*core.MapModification, 0),
			Removed:  make([]*core.MapChange, 0),
		}
		var v typegen.Deferred
		if err := currentMap.ForEach(&v, func(key string) error {
			out.Added = append(out.Added, &core.MapChange{
				Key:   key,
				Value: v,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}

	executedActor, err := actorStateLoader(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}

	executedMap, executedMapOpts, err := actorMapLoader(executedActor)
	if err != nil {
		return nil, err
	}

	return core.DiffMap(ctx, api.Store(), currentMap, executedMap, currentMapOpts, executedMapOpts)
}
