package v2

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v2"
)

func MinerHandler(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, minerMapRoot cid.Cid) (model.PersistableList, error) {
	out := model.PersistableList{}
	adtStore := store.WrapBlockStore(ctx, bs)
	minerMap, err := adt.AsMap(adtStore, minerMapRoot, 5)
	if err != nil {
		return nil, err
	}
	minerState := new(v2.StateChange)
	if err := minerMap.ForEach(minerState, func(key string) error {
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		minerStateDiff, err := minerState.StateChangeAsStateDiffResult(ctx, bs)
		if err != nil {
			return err
		}
		stateExtractor := &StateExtract{
			ExtractMethods: []Extractor{
				// TODO
			},
		}
		models, err := stateExtractor.Process(ctx, current, executed, addr, minerStateDiff)
		if err != nil {
			return err
		}
		out = append(out, models...)
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type Extractor interface {
	Extract(ctx context.Context, current, parent *types.TipSet, addr address.Address, change *v2.StateDiffResult) (model.Persistable, error)
}

type StateExtract struct {
	ExtractMethods []Extractor
}

func (se *StateExtract) Process(ctx context.Context, current, executed *types.TipSet, addr address.Address, changes *v2.StateDiffResult) (model.PersistableList, error) {
	out := make(model.PersistableList, 0, len(se.ExtractMethods))
	for _, e := range se.ExtractMethods {
		m, err := e.Extract(ctx, current, executed, addr, changes)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
