package v2

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
	v2_9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v2/v9"
)

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, change *verifregdiff.StateDiffResult) (model.Persistable, error)
}

func TransformVerifregState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid, transformers ...Transformer) (model.Persistable, error) {
	transformers, err := LookupVerifregStateTransformer(version)
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

func LookupVerifregStateTransformer(av actortypes.Version) ([]Transformer, error) {
	switch av {
	case actortypes.Version9:
		return []Transformer{
			v2_9.Verifiers{},
			//v2_9.Claims{},
			//v2_9.Allocations{},
		}, nil
	}
	return nil, fmt.Errorf("unsupported actor version for verifreg transform: %d", av)
}
