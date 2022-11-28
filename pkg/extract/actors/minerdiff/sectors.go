package minerdiff

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

var _ ActorStateChange = (*SectorChangeList)(nil)

type SectorChange struct {
	Sector typegen.Deferred
	Type   core.ChangeType
}

type SectorChangeList []*SectorChange

func (s SectorChangeList) Kind() ActorStateKind {
	return "miner_sector"
}

type Sectors struct{}

func (Sectors) Diff(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error) {
	return DiffSectors(ctx, api, act, current, executed)
}

func DiffSectors(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error) {
	// the actor was removed, nothing has changes in the current state.
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}

	currentMiner, err := miner.Load(api.Store(), act.Actor)
	if err != nil {
		return nil, err
	}

	// the actor was added, everything is new in the current state.
	if act.Type == core.ChangeTypeAdd {
		sa, err := currentMiner.SectorsArray()
		if err != nil {
			return nil, err
		}
		out := make(SectorChangeList, int(sa.Length()))
		var v typegen.Deferred
		if err := sa.ForEach(&v, func(idx int64) error {
			out[idx] = &SectorChange{
				Sector: v,
				Type:   core.ChangeTypeAdd,
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}

	// the actor was modified, diff against executed state.
	executedActor, err := api.Actor(ctx, act.Address, executed.Key())
	if err != nil {
		return nil, err
	}
	executedMiner, err := miner.Load(api.Store(), executedActor)
	if err != nil {
		return nil, err
	}

	sectorChanges, err := miner.DiffSectorsDeferred(ctx, api.Store(), executedMiner, currentMiner)
	if err != nil {
		return nil, err
	}

	idx := 0
	out := make(SectorChangeList, len(sectorChanges.Added)+len(sectorChanges.Removed)+len(sectorChanges.Modified))
	for _, change := range sectorChanges.Added {
		out[idx] = &SectorChange{
			Sector: *change,
			Type:   core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range sectorChanges.Removed {
		out[idx] = &SectorChange{
			Sector: *change,
			Type:   core.ChangeTypeRemove,
		}
		idx++
	}
	for _, change := range sectorChanges.Modified {
		out[idx] = &SectorChange{
			Sector: *change.Current,
			Type:   core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil

}
