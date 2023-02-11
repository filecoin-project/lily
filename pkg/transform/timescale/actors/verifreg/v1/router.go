package v1

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	verifregdiff "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
	v1_0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v0"
	v1_2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v2"
	v1_3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v3"
	v1_4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v4"
	v1_5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v5"
	v1_6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v6"
	v1_7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v7"
	v1_8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/verifreg/v1/v8"
)

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, change *verifregdiff.StateDiffResult) (model.Persistable, error)
}

func TransformVerifregState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid) (model.Persistable, error) {
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
	case actortypes.Version0:
		return []Transformer{
			v1_0.Clients{},
			v1_0.Verifiers{},
		}, nil
	case actortypes.Version2:
		return []Transformer{
			v1_2.Clients{},
			v1_2.Verifiers{},
		}, nil
	case actortypes.Version3:
		return []Transformer{
			v1_3.Clients{},
			v1_3.Verifiers{},
		}, nil
	case actortypes.Version4:
		return []Transformer{
			v1_4.Clients{},
			v1_4.Verifiers{},
		}, nil
	case actortypes.Version5:
		return []Transformer{
			v1_5.Clients{},
			v1_5.Verifiers{},
		}, nil
	case actortypes.Version6:
		return []Transformer{
			v1_6.Clients{},
			v1_6.Verifiers{},
		}, nil
	case actortypes.Version7:
		return []Transformer{
			v1_7.Clients{},
			v1_7.Verifiers{},
		}, nil
	case actortypes.Version8:
		return []Transformer{
			v1_8.Clients{},
			v1_8.Verifiers{},
		}, nil
	}
	return nil, fmt.Errorf("unsupported actor version for verifreg transform: %d", av)
}
