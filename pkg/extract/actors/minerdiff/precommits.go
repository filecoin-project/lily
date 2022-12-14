package minerdiff

import (
	"context"
	"time"

	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*PreCommitChangeList)(nil)

type PreCommitChange struct {
	SectorNumber []byte            `cborgen:"sector_number"`
	Current      *typegen.Deferred `cborgen:"current_pre_commit"`
	Previous     *typegen.Deferred `cborgen:"previous_pre_commit"`
	Change       core.ChangeType   `cborgen:"change"`
}

type PreCommitChangeList []*PreCommitChange

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
