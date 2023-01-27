package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/model"
	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	v1_0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v0"
	v1_2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v2"
	v1_3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v3"
	v1_4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v4"
	v1_5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v5"
	v1_6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v6"
	v1_7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v7"
	v1_8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v8"
	v1_9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/v9"
)

func TransformMinerStates(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root cid.Cid) (model.Persistable, error) {
	// a map of all miners whose state has changes
	minerMap, err := adt.AsMap(s, root, 5)
	if err != nil {
		return nil, err
	}

	// load map entires into a list to process below
	var miners []*minertypes.MinerStateChange
	minerState := new(minerdiff.StateChange)
	if err := minerMap.ForEach(minerState, func(key string) error {
		// the map is keyed by the miner address, the value of the map contains the miners state change
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		// decode the miner state change from the IPLD structure to a type we can inspect.
		minerStateDiff, err := minerState.ToStateDiffResult(ctx, s)
		if err != nil {
			return err
		}
		miners = append(miners, &minertypes.MinerStateChange{
			Address:     addr,
			StateChange: minerStateDiff,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	// get a list of transformers capable of handling this actor version
	transformers, err := LookupMinerStateTransformer(version)
	if err != nil {
		return nil, err
	}

	// run each transformer keeping its model
	out := model.PersistableList{}
	for _, t := range transformers {
		m := t.Transform(ctx, current, executed, miners)
		out = append(out, m)
	}
	return out, nil
}

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable
}

func LookupMinerStateTransformer(av actortypes.Version) ([]Transformer, error) {
	switch av {
	case actortypes.Version0:
		return []Transformer{
			v1_0.Info{},
			v1_0.PreCommit{},
			v1_0.SectorDeal{},
			v1_0.SectorEvent{},
			v1_0.Sector{},
		}, nil
	case actortypes.Version2:
		return []Transformer{
			v1_2.Info{},
			v1_2.PreCommit{},
			v1_2.SectorDeal{},
			v1_2.SectorEvent{},
			v1_2.Sector{},
		}, nil
	case actortypes.Version3:
		return []Transformer{
			v1_3.Info{},
			v1_3.PreCommit{},
			v1_3.SectorDeal{},
			v1_3.SectorEvent{},
			v1_3.Sector{},
		}, nil
	case actortypes.Version4:
		return []Transformer{
			v1_4.Info{},
			v1_4.PreCommit{},
			v1_4.SectorDeal{},
			v1_4.SectorEvent{},
			v1_4.Sector{},
		}, nil
	case actortypes.Version5:
		return []Transformer{
			v1_5.Info{},
			v1_5.PreCommit{},
			v1_5.SectorDeal{},
			v1_5.SectorEvent{},
			v1_5.Sector{},
		}, nil
	case actortypes.Version6:
		return []Transformer{
			v1_6.Info{},
			v1_6.PreCommit{},
			v1_6.SectorDeal{},
			v1_6.SectorEvent{},
			v1_6.Sector{},
		}, nil
	case actortypes.Version7:
		return []Transformer{
			v1_7.Info{},
			v1_7.PreCommit{},
			v1_7.SectorDeal{},
			v1_7.SectorEvent{},
			v1_7.Sector{},
		}, nil
	case actortypes.Version8:
		return []Transformer{
			v1_8.Info{},
			v1_8.PreCommit{},
			v1_8.SectorDeal{},
			v1_8.SectorEvent{},
			v1_8.Sector{},
		}, nil
	case actortypes.Version9:
		return []Transformer{
			v1_9.Info{},
			v1_9.PreCommit{},
			v1_9.SectorDeal{},
			v1_9.SectorEvent{},
			v1_9.Sector{},
		}, nil

	}
	return nil, fmt.Errorf("unsupported actor version for miner transform: %d", av)
}
