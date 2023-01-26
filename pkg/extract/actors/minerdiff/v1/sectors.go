package v1

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

type SectorChange struct {
	SectorNumber uint64            `cborgen:"sector_number"`
	Current      *typegen.Deferred `cborgen:"current_sector"`
	Previous     *typegen.Deferred `cborgen:"previous_sector"`
	Change       core.ChangeType   `cborgen:"change"`
}

type SectorChangeList []*SectorChange

func (s SectorChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range s {
		if err := node.Put(abi.UIntKey(l.SectorNumber), l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
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
