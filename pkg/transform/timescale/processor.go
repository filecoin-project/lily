package timescale

import (
	"context"
	"fmt"
	"io"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	v1car "github.com/ipld/go-car"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/transform/cbor"
	cborminer "github.com/filecoin-project/lily/pkg/transform/cbor/miner"
	"github.com/filecoin-project/lily/pkg/transform/timescale/miner"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/miner/v9"
)

func Process(ctx context.Context, r io.Reader, strg model.Storage) error {
	bs := blockstore.NewMemorySync()
	header, err := v1car.LoadCar(ctx, bs, r)
	if err != nil {
		return err
	}
	if len(header.Roots) != 1 {
		return fmt.Errorf("invalid header expected 1 root got %d", len(header.Roots))
	}

	store := store.WrapBlockStore(ctx, bs)

	var actorIPLDContainer cbor.ActorIPLDContainer
	if err := store.Get(ctx, header.Roots[0], &actorIPLDContainer); err != nil {
		return err
	}

	minerMap, err := adt.AsMap(store, actorIPLDContainer.MinerActors, 5)
	if err != nil {
		return err
	}

	var minerState cborminer.StateChange
	var minerModels model.PersistableList
	if err := minerMap.ForEach(&minerState, func(key string) error {
		stateChange, err := miner.DecodeMinerStateDiff(ctx, store, minerState)
		if err != nil {
			return err
		}
		minerModel, err := v9.ProcessMinerStateChanges(ctx, store, actorIPLDContainer.CurrentTipSet, actorIPLDContainer.ExecutedTipSet, minerState.Miner, stateChange)
		if err != nil {
			return err
		}
		minerModels = append(minerModels, minerModel...)
		return nil
	}); err != nil {
		return err
	}

	return strg.PersistBatch(ctx, minerModels...)
}
