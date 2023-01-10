package v0

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*PreCommitChangeList)(nil)

var _ abi.Keyer = (*PreCommitChange)(nil)

type PreCommitChange struct {
	SectorNumber []byte            `cborgen:"sector_number"`
	Current      *typegen.Deferred `cborgen:"current_pre_commit"`
	Previous     *typegen.Deferred `cborgen:"previous_pre_commit"`
	Change       core.ChangeType   `cborgen:"change"`
}

func (t *PreCommitChange) Key() string {
	return core.StringKey(t.SectorNumber).Key()
}

type PreCommitChangeList []*PreCommitChange

func (p PreCommitChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range p {
		if err := node.Put(l, l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

func (p *PreCommitChangeList) FromAdtMap(store adt.Store, root cid.Cid, bw int) error {
	precommitMap, err := adt.AsMap(store, root, bw)
	if err != nil {
		return err
	}

	precommits := new(PreCommitChangeList)
	precommitChange := new(PreCommitChange)
	if err := precommitMap.ForEach(precommitChange, func(sectorNumber string) error {
		val := new(PreCommitChange)
		*val = *precommitChange
		*precommits = append(*precommits, val)
		return nil
	}); err != nil {
		return err
	}
	*p = *precommits
	return nil

}

const KindMinerPreCommit = "miner_precommit"

func (p PreCommitChangeList) Kind() actors.ActorStateKind {
	return KindMinerPreCommit
}

type PreCommit struct{}

func (PreCommit) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerPreCommit, zap.Inline(act), "duration", time.Since(start))
	}()
	return PreCommitDiff(ctx, api, act)
}

func PreCommitDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, MinerStateLoader, MinerPreCommitMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(PreCommitChangeList, len(mapChange))
	for i, change := range mapChange {
		out[i] = &PreCommitChange{
			SectorNumber: change.Key,
			Current:      change.Current,
			Previous:     change.Previous,
			Change:       change.Type,
		}
	}
	return out, nil
}
