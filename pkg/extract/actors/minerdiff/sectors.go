package minerdiff

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*SectorChangeList)(nil)

type SectorChange struct {
	// TODO include sectorID key
	Sector typegen.Deferred `cborgen:"sector"`
	Change core.ChangeType  `cborgen:"change"`
}

type SectorChangeList []*SectorChange

const KindMinerSector = "miner_sector"

func (s SectorChangeList) Kind() actors.ActorStateKind {
	return KindMinerSector
}

type Sectors struct{}

func (Sectors) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return DiffSectors(ctx, api, act)
}

func DiffSectors(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	arrayChange, err := generic.DiffActorArray(ctx, api, act, MinerStateLoader, MinerSectorArrayLoader)
	if err != nil {
		return nil, err
	}
	idx := 0
	out := make(SectorChangeList, arrayChange.Size())
	for _, change := range arrayChange.Added {
		out[idx] = &SectorChange{
			Sector: change.Value,
			Change: core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range arrayChange.Removed {
		out[idx] = &SectorChange{
			Sector: change.Value,
			Change: core.ChangeTypeRemove,
		}
		idx++
	}
	for _, change := range arrayChange.Modified {
		out[idx] = &SectorChange{
			Sector: change.Current,
			Change: core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil
}
