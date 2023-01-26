package market

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	marketdiff "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v0"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v0"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v2"
	v3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v3"
	v4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v4"
	v5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v5"
	v6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v6"
	v7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v7"
	v8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v8"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/market/v9"
)

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, change *marketdiff.StateDiffResult) (model.Persistable, error)
}

func TransformMarketState(ctx context.Context, s store.Store, current, executed *types.TipSet, root cid.Cid, transformers ...Transformer) (model.Persistable, error) {
	marketState := new(marketdiff.StateChange)
	if err := s.Get(ctx, root, marketState); err != nil {
		return nil, err
	}
	marketStateDiff, err := marketState.ToStateDiffResult(ctx, s)
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
			v0.Deals{},
			v0.Proposals{},
		}, nil
	case actortypes.Version2:
		return []Transformer{
			v2.Deals{},
			v2.Proposals{},
		}, nil
	case actortypes.Version3:
		return []Transformer{
			v3.Deals{},
			v3.Proposals{},
		}, nil
	case actortypes.Version4:
		return []Transformer{
			v4.Deals{},
			v4.Proposals{},
		}, nil
	case actortypes.Version5:
		return []Transformer{
			v5.Deals{},
			v5.Proposals{},
		}, nil
	case actortypes.Version6:
		return []Transformer{
			v6.Deals{},
			v6.Proposals{},
		}, nil
	case actortypes.Version7:
		return []Transformer{
			v7.Deals{},
			v7.Proposals{},
		}, nil
	case actortypes.Version8:
		return []Transformer{
			v8.Deals{},
			v8.Proposals{},
		}, nil
	case actortypes.Version9:
		return []Transformer{
			v9.Deals{},
			v9.Proposals{},
		}, nil
	}
	return nil, fmt.Errorf("unsupported actor version for market transform: %d", av)
}
