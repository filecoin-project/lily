package generic

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

func DiffActorArray(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, actorStateLoader ActorStateLoader, actorArrayLoader ActorStateArrayLoader) (core.ArrayModifications, error) {
	if act.Type == core.ChangeTypeRemove {
		executedActor, err := actorStateLoader(api.Store(), act.Executed)
		if err != nil {
			return nil, err
		}

		executedArray, _, err := actorArrayLoader(executedActor)
		if err != nil {
			return nil, err
		}

		var out core.ArrayModifications
		var v typegen.Deferred
		if err := executedArray.ForEach(&v, func(key int64) error {
			value := v
			out = append(out, &core.ArrayModification{
				Key:      uint64(key),
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

	currentArray, currentArrayBitWidth, err := actorArrayLoader(currentActor)
	if err != nil {
		return nil, err
	}
	if act.Type == core.ChangeTypeAdd {
		var out core.ArrayModifications
		var v typegen.Deferred
		if err := currentArray.ForEach(&v, func(key int64) error {
			value := v
			out = append(out, &core.ArrayModification{
				Key:      uint64(key),
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

	executedArray, executedArrayBitWidth, err := actorArrayLoader(executedActor)
	if err != nil {
		return nil, err
	}

	return core.DiffArray(ctx, api.Store(), currentArray, executedArray, currentArrayBitWidth, executedArrayBitWidth)
}
