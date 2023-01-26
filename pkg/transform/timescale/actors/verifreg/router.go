package verifreg

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v0"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v9"
)

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, change *verifregdiff.StateDiffResult) (model.Persistable, error)
}

func TransformVerifregState(ctx context.Context, s store.Store, av actortypes.Version, current, executed *types.TipSet, root cid.Cid, transformers ...Transformer) (model.Persistable, error) {
	if av < actortypes.Version9 {
		verifregState := new(verifregdiff.StateChange)
		if err := s.Get(ctx, root, verifregState); err != nil {
			return nil, err
		}
		verifrefStateDiff, err := verifregState.ToStateDiffResult(ctx, s)
		if err != nil {
			return nil, err
		}
		out := model.PersistableList{}
		for _, t := range transformers {
			m, err := t.Transform(ctx, current, executed, verifrefStateDiff)
			if err != nil {
				return nil, err
			}
			out = append(out, m)
		}
		return out, nil

	}
}

func LookupMarketStateTransformerV8(av actortypes.Version) ([]Transformer, error) {
	switch av {
	case actortypes.Version0:
		return []Transformer{
			v0.Clients{},
			v0.Verifiers{},
		}, nil
	case actortypes.Version2:
		return []Transformer{}, nil
	case actortypes.Version3:
		return []Transformer{}, nil
	case actortypes.Version4:
		return []Transformer{}, nil
	case actortypes.Version5:
		return []Transformer{}, nil
	case actortypes.Version6:
		return []Transformer{}, nil
	case actortypes.Version7:
		return []Transformer{}, nil
	case actortypes.Version8:
		return []Transformer{}, nil
	case actortypes.Version9:
		return []Transformer{
			v9.Verifiers{},
		}, nil
	}
	return nil, fmt.Errorf("unsupported actor version for market transform: %d", av)
}
