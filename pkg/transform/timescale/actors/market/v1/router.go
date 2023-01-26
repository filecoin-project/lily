package v1

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v0"
	v1_0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v0"
	v1_2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v2"
	v1_3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v3"
	v1_4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v4"
	v1_5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v5"
	v1_6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v6"
	v1_7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v7"
	v1_8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v8"
	v1_9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v1/v9"
)

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, change *marketdiff.StateDiffResult) (model.Persistable, error)
}

func TransformMarketState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid) (model.Persistable, error) {
	marketState := new(marketdiff.StateChange)
	if err := s.Get(ctx, root, marketState); err != nil {
		return nil, err
	}
	marketStateDiff, err := marketState.ToStateDiffResult(ctx, s)
	if err != nil {
		return nil, err
	}
	transformers, err := LookupMarketStateTransformer(version)
	if err != nil {
		return nil, err
	}
	out := model.PersistableList{}
	for _, t := range transformers {
		m, err := t.Transform(ctx, current, executed, marketStateDiff)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func LookupMarketStateTransformer(av actortypes.Version) ([]Transformer, error) {
	switch av {
	case actortypes.Version0:
		return []Transformer{
			v1_0.Deals{},
			v1_0.Proposals{},
		}, nil
	case actortypes.Version2:
		return []Transformer{
			v1_2.Deals{},
			v1_2.Proposals{},
		}, nil
	case actortypes.Version3:
		return []Transformer{
			v1_3.Deals{},
			v1_3.Proposals{},
		}, nil
	case actortypes.Version4:
		return []Transformer{
			v1_4.Deals{},
			v1_4.Proposals{},
		}, nil
	case actortypes.Version5:
		return []Transformer{
			v1_5.Deals{},
			v1_5.Proposals{},
		}, nil
	case actortypes.Version6:
		return []Transformer{
			v1_6.Deals{},
			v1_6.Proposals{},
		}, nil
	case actortypes.Version7:
		return []Transformer{
			v1_7.Deals{},
			v1_7.Proposals{},
		}, nil
	case actortypes.Version8:
		return []Transformer{
			v1_8.Deals{},
			v1_8.Proposals{},
		}, nil
	case actortypes.Version9:
		return []Transformer{
			v1_9.Deals{},
			v1_9.Proposals{},
		}, nil
	}
	return nil, fmt.Errorf("unsupported actor version for market transform: %d", av)
}
