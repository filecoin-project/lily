package v0

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*SectorChangeList)(nil)

var _ abi.Keyer = (*SectorChange)(nil)

type SectorChange struct {
	SectorNumber uint64            `cborgen:"sector_number"`
	Current      *typegen.Deferred `cborgen:"current_sector"`
	Previous     *typegen.Deferred `cborgen:"previous_sector"`
	Change       core.ChangeType   `cborgen:"change"`
}

func (t *SectorChange) Key() string {
	return abi.UIntKey(t.SectorNumber).Key()
}

type SectorChangeList []*SectorChange

func (s SectorChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range s {
		if err := node.Put(l, l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

func (s *SectorChangeList) FromAdtMap(store adt.Store, root cid.Cid, bw int) error {
	sectorMap, err := adt.AsMap(store, root, 5)
	if err != nil {
		return err
	}

	sectors := new(SectorChangeList)
	sectorChange := new(SectorChange)
	if err := sectorMap.ForEach(sectorChange, func(sectorNumber string) error {
		val := new(SectorChange)
		*val = *sectorChange
		*sectors = append(*sectors, val)
		return nil
	}); err != nil {
		return err
	}
	*s = *sectors
	return nil
}

const KindMinerSector = "miner_sector"

func (s SectorChangeList) Kind() actors.ActorStateKind {
	return KindMinerSector
}

type Sectors struct{}

func (Sectors) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerSector, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffSectors(ctx, api, act)
}

func DiffSectors(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	arrayChange, err := generic.DiffActorArray(ctx, api, act, MinerStateLoader, MinerSectorArrayLoader)
	if err != nil {
		return nil, err
	}
	out := make(SectorChangeList, len(arrayChange))
	for i, change := range arrayChange {
		out[i] = &SectorChange{
			SectorNumber: change.Key,
			Current:      change.Current,
			Previous:     change.Previous,
			Change:       change.Type,
		}
	}
	return out, nil
}
