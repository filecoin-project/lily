package minerdiff

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*SectorChangeList)(nil)

type SectorChange struct {
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
	minerStateLoader := func(store adt.Store, act *types.Actor) (interface{}, error) {
		return miner.Load(api.Store(), act)
	}
	minerArrayLoader := func(m interface{}) (adt.Array, int, error) {
		minerState := m.(miner.State)
		sectorArray, err := minerState.SectorsArray()
		if err != nil {
			return nil, -1, err
		}
		return sectorArray, minerState.SectorsAmtBitwidth(), nil
	}
	arrayChange, err := generic.DiffActorArray(ctx, api, act, minerStateLoader, minerArrayLoader)
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
