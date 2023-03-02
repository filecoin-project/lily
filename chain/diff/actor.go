package diff

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type ActorStateLoader = func(adt.Store, *types.Actor) (interface{}, error)
type ActorStateMapLoader = func(interface{}) (adt.Map, *adt.MapOpts, error)

func DiffActorMap(ctx context.Context, api actorstate.ActorStateAPI, act actorstate.ActorInfo, actorStateLoader ActorStateLoader, actorMapLoader ActorStateMapLoader) (MapModifications, error) {
	if act.ChangeType == tasks.ChangeTypeRemove {
		prevActor, err := api.Actor(ctx, act.Address, act.Executed.Key())
		if err != nil {
			return nil, err
		}
		executedActor, err := actorStateLoader(api.Store(), prevActor)
		if err != nil {
			return nil, err
		}

		executedMap, _, err := actorMapLoader(executedActor)
		if err != nil {
			return nil, err
		}

		var out MapModifications
		var v typegen.Deferred
		if err := executedMap.ForEach(&v, func(key string) error {
			value := v
			out = append(out, &MapModification{
				Key:      []byte(key),
				Type:     tasks.ChangeTypeRemove,
				Previous: &value,
				Current:  nil,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}

	currentActor, err := actorStateLoader(api.Store(), &act.Actor)
	if err != nil {
		return nil, err
	}

	currentMap, currentMapOpts, err := actorMapLoader(currentActor)
	if err != nil {
		return nil, err
	}

	if act.ChangeType == tasks.ChangeTypeAdd {
		var out MapModifications
		var v typegen.Deferred
		if err := currentMap.ForEach(&v, func(key string) error {
			value := v
			out = append(out, &MapModification{
				Key:      []byte(key),
				Type:     tasks.ChangeTypeAdd,
				Previous: nil,
				Current:  &value,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil

	}
	prevActor, err := api.Actor(ctx, act.Address, act.Executed.Key())
	if err != nil {
		return nil, err
	}

	executedActor, err := actorStateLoader(api.Store(), prevActor)
	if err != nil {
		return nil, err
	}

	executedMap, executedMapOpts, err := actorMapLoader(executedActor)
	if err != nil {
		return nil, err
	}

	return DiffMap(ctx, api.Store(), currentMap, executedMap, currentMapOpts, executedMapOpts)
}
