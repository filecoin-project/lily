package miner

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
	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	v0 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v0"
	v2 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v2"
	v3 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v3"
	v4 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v4"
	v5 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v5"
	v6 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v6"
	v7 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v7"
	v8 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v8"
	v9 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v9"
)

func HandleMiner(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *rawdiff.ActorChange, version actortypes.Version) (model.Persistable, error) {
	switch version {
	case actortypes.Version0:
		return v0.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version2:
		return v2.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version3:
		return v3.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version4:
		return v4.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version5:
		return v5.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version6:
		return v6.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version7:
		return v7.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version8:
		return v8.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version9:
		return v9.ExtractMinerStateChanges(ctx, current, executed, addr, change)
	case actortypes.Version10:
		panic("not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported miner actor version: %d", version)
	}
}

func TransformMinerStates(ctx context.Context, s store.Store, current, executed *types.TipSet, root cid.Cid, transformers ...Transformer) (model.Persistable, error) {
	out := model.PersistableList{}

	// a map of all miners whose state has changes
	minerMap, err := adt.AsMap(s, root, 5)
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
		minerStateDiff, err := minerState.ToStateDiffResult(ctx, s)
		if err != nil {
			return err
		}
		for _, t := range transformers {
			m, err := t.Transform(ctx, current, executed, addr, minerStateDiff)
			if err != nil {
				return err
			}
			out = append(out, m)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type Transformer interface {
	Transform(ctx context.Context, current, parent *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error)
}

func LookupMinerStateTransformer(av actortypes.Version) ([]Transformer, error) {
	switch av {
	case actortypes.Version0:
		return []Transformer{
			v0.Info{},
			v0.PreCommit{},
			v0.SectorDeal{},
			v0.SectorEvent{},
			v0.Sector{},
		}, nil
	case actortypes.Version2:
		return []Transformer{
			v2.Info{},
			v2.PreCommit{},
			v2.SectorDeal{},
			v2.SectorEvent{},
			v2.Sector{},
		}, nil
	case actortypes.Version3:
		return []Transformer{
			v3.Info{},
			v3.PreCommit{},
			v3.SectorDeal{},
			v3.SectorEvent{},
			v3.Sector{},
		}, nil
	case actortypes.Version4:
		return []Transformer{
			v4.Info{},
			v4.PreCommit{},
			v4.SectorDeal{},
			v4.SectorEvent{},
			v4.Sector{},
		}, nil
	case actortypes.Version5:
		return []Transformer{
			v5.Info{},
			v5.PreCommit{},
			v5.SectorDeal{},
			v5.SectorEvent{},
			v5.Sector{},
		}, nil
	case actortypes.Version6:
		return []Transformer{
			v6.Info{},
			v6.PreCommit{},
			v6.SectorDeal{},
			v6.SectorEvent{},
			v6.Sector{},
		}, nil
	case actortypes.Version7:
		return []Transformer{
			v7.Info{},
			v7.PreCommit{},
			v7.SectorDeal{},
			v7.SectorEvent{},
			v7.Sector{},
		}, nil
	case actortypes.Version8:
		return []Transformer{
			v8.Info{},
			v8.PreCommit{},
			v8.SectorDeal{},
			v8.SectorEvent{},
			v8.Sector{},
		}, nil
	case actortypes.Version9:
		return []Transformer{
			v9.Info{},
			v9.PreCommit{},
			v9.SectorDeal{},
			v9.SectorEvent{},
			v9.Sector{},
		}, nil

	}
	return nil, fmt.Errorf("unsupported actor version for miner transform: %d", av)
}
