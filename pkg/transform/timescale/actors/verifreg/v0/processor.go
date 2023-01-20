package v0

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
)

type Extractor interface {
	Extract(ctx context.Context, current, executed *types.TipSet, change *verifregdiff.StateDiffResult) (model.Persistable, error)
}

type StateExtract struct {
	Extractors []Extractor
}

func (se *StateExtract) Process(ctx context.Context, current, executed *types.TipSet, changes *verifregdiff.StateDiffResult) (model.PersistableList, error) {
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

func VerifregHandler(ctx context.Context, s store.Store, current, executed *types.TipSet, verifregRoot cid.Cid) (model.PersistableList, error) {
	verifregState := new(verifregdiff.StateChange)
	if err := s.Get(ctx, verifregRoot, verifregState); err != nil {
		return nil, err
	}
	verifrefStateDiff, err := verifregState.ToStateDiffResult(ctx, s)
	if err != nil {
		return nil, err
	}

	stateExtractor := &StateExtract{
		Extractors: []Extractor{
			Clients{},
			Verifiers{},
		},
	}
	return stateExtractor.Process(ctx, current, executed, verifrefStateDiff)
}
