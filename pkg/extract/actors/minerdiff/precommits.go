package minerdiff

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*PreCommitChangeList)(nil)

type PreCommitChange struct {
	// TODO include sector ID
	PreCommit typegen.Deferred `cborgen:"pre_commit"`
	Change    core.ChangeType  `cborgen:"change"`
}

type PreCommitChangeList []*PreCommitChange

const KindMinerPreCommit = "miner_precommit"

func (p PreCommitChangeList) Kind() actors.ActorStateKind {
	return KindMinerPreCommit
}

type PreCommit struct{}

func (PreCommit) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return PreCommitDiff(ctx, api, act)
}

func PreCommitDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	mapChange, err := generic.DiffActorMap(ctx, api, act, MinerStateLoader, MinerPreCommitMapLoader)
	if err != nil {
		return nil, err
	}
	out := make(PreCommitChangeList, mapChange.Size())
	idx := 0
	for _, change := range mapChange.Added {
		out[idx] = &PreCommitChange{
			PreCommit: change.Value,
			Change:    core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range mapChange.Removed {
		out[idx] = &PreCommitChange{
			PreCommit: change.Value,
			Change:    core.ChangeTypeRemove,
		}
		idx++
	}
	// NB: PreCommits cannot be modified, but check anyway.
	for _, change := range mapChange.Modified {
		out[idx] = &PreCommitChange{
			PreCommit: change.Current,
			Change:    core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil
}
