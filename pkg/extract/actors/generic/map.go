package generic

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

func DiffActorMap(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, actorStateLoader ActorStateLoader, actorMapLoader ActorStateMapLoader) (core.MapModifications, error) {
	if act.Type == core.ChangeTypeRemove {
		executedActor, err := actorStateLoader(api.Store(), act.Executed)
		if err != nil {
			return nil, err
		}

		executedMap, _, err := actorMapLoader(executedActor)
		if err != nil {
			return nil, err
		}

		var out core.MapModifications
		var v typegen.Deferred
		if err := executedMap.ForEach(&v, func(key string) error {
			value := v
			out = append(out, &core.MapModification{
				Key:      []byte(key),
				Type:     core.ChangeTypeRemove,
				Previous: &value,
				Current:  nil,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
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
		var out core.MapModifications
		var v typegen.Deferred
		if err := currentMap.ForEach(&v, func(key string) error {
			value := v
			out = append(out, &core.MapModification{
				Key:      []byte(key),
				Type:     core.ChangeTypeAdd,
				Previous: nil,
				Current:  &value,
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
