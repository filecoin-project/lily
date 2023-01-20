package v0

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v0"
)

type Extractor interface {
	Extract(ctx context.Context, current, executed *types.TipSet, change *marketdiff.StateDiffResult) (model.Persistable, error)
}

type StateExtract struct {
	Extractors []Extractor
}

func (se *StateExtract) Process(ctx context.Context, current, executed *types.TipSet, changes *marketdiff.StateDiffResult) (model.PersistableList, error) {
	out := make(model.PersistableList, 0, len(se.Extractors))
	for _, e := range se.Extractors {
		m, err := e.Extract(ctx, current, executed, changes)
		if err != nil {
			return nil, err
		}
		if m != nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func MarketHandler(ctx context.Context, s store.Store, current, executed *types.TipSet, marketRoot cid.Cid) (model.PersistableList, error) {
	marketState := new(marketdiff.StateChange)
	if err := s.Get(ctx, marketRoot, marketState); err != nil {
		return nil, err
	}
	marketStateDiff, err := marketState.ToStateDiffResult(ctx, s)
	if err != nil {
		return nil, err
	}

	stateExtractor := &StateExtract{
		Extractors: []Extractor{
			Deals{},
			Proposals{},
		},
	}
	return stateExtractor.Process(ctx, current, executed, marketStateDiff)
}
