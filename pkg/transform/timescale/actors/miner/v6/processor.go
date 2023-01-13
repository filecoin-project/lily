package v6

import (
	"context"

	"github.com/filecoin-project/go-address"
	store "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v6"
)

type Extractor interface {
	Extract(ctx context.Context, current, parent *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error)
}

type StateExtract struct {
	ExtractMethods []Extractor
}

func (se *StateExtract) Process(ctx context.Context, current, executed *types.TipSet, addr address.Address, changes *minerdiff.StateDiffResult) (model.PersistableList, error) {
	out := make(model.PersistableList, 0, len(se.ExtractMethods))
	for _, e := range se.ExtractMethods {
		m, err := e.Extract(ctx, current, executed, addr, changes)
		if err != nil {
			return nil, err
		}
		if m != nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func MinerStateHandler(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange) (model.Persistable, error) {
	return ExtractMinerStateChanges(ctx, current, executed, addr, change)
}

func MinerHandler(ctx context.Context, bs blockstore.Blockstore, current, executed *types.TipSet, minerMapRoot cid.Cid) (model.PersistableList, error) {
	out := model.PersistableList{}
	adtStore := store.WrapBlockStore(ctx, bs)

	// a map of all miners whose state has changes
	minerMap, err := adt.AsMap(adtStore, minerMapRoot, 5)
	if err != nil {
		return nil, err
	}
	// iterate over each miner with a state change
	minerState := new(minerdiff.StateChange)
	if err := minerMap.ForEach(minerState, func(key string) error {
		// the map is keyed by the miner address, the value of the map contains the miners state change
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		// decode the miner state change from the IPLD structure to a type we can inspect.
		minerStateDiff, err := minerdiff.DecodeStateDiffResultFromStateChange(ctx, bs, minerState)
		if err != nil {
			return err
		}
		// register extractors to run over the miners state.
		stateExtractor := &StateExtract{
			ExtractMethods: []Extractor{
				Info{},
				PreCommit{},
				SectorDeal{},
				SectorEvent{},
				Sector{},
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
