package generic

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type ActorStateArrayLoader = func(interface{}) (adt.Array, int, error)

type ArrayChange struct {
	Key    uint64
	Value  typegen.Deferred
	Change core.ChangeType
}

type ArrayChangeList = []*MapChange

func DiffActorArray(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, actorStateLoader ActorStateLoader, actorArrayLoader ActorStateArrayLoader) (*core.ArrayDiff, error) {
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
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
		out := &core.ArrayDiff{
			Added:    make([]*core.ArrayChange, 0),
			Modified: make([]*core.ArrayModification, 0),
			Removed:  make([]*core.ArrayChange, 0),
		}
		var v typegen.Deferred
		if err := currentArray.ForEach(&v, func(key int64) error {
			out.Added = append(out.Added, &core.ArrayChange{
				// TODO this type is inconsistent in specs-actors..
				Key:   uint64(key),
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

	executedArray, executedArrayBitWidth, err := actorArrayLoader(executedActor)
	if err != nil {
		return nil, err
	}

	return core.DiffArray(ctx, api.Store(), currentArray, executedArray, currentArrayBitWidth, executedArrayBitWidth)
}
