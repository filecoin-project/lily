package minerdiff

import (
	"context"

	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*PreCommitChangeList)(nil)

type PreCommitChange struct {
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
	return DiffPreCommits(ctx, api, act)
}

func DiffPreCommits(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	// the actor was removed, nothing has changes in the current state.
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}

	currentMiner, err := miner.Load(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}
	// the actor was added, everything is new in the current state.
	if act.Type == core.ChangeTypeAdd {
		var out PreCommitChangeList
		pm, err := currentMiner.PrecommitsMap()
		if err != nil {
			return nil, err
		}
		var v typegen.Deferred
		if err := pm.ForEach(&v, func(key string) error {
			out = append(out, &PreCommitChange{
				PreCommit: v,
				Change:    core.ChangeTypeAdd,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}

	// actor was modified load executed state.
	executedMiner, err := miner.Load(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}

	preCommitChanges, err := miner.DiffPreCommitsDeferred(ctx, api.Store(), executedMiner, currentMiner)
	if err != nil {
		return nil, err
	}

	idx := 0
	out := make(PreCommitChangeList, len(preCommitChanges.Added)+len(preCommitChanges.Removed)+len(preCommitChanges.Modified))
	for _, change := range preCommitChanges.Added {
		out[idx] = &PreCommitChange{
			PreCommit: *change,
			Change:    core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range preCommitChanges.Removed {
		out[idx] = &PreCommitChange{
			PreCommit: *change,
			Change:    core.ChangeTypeRemove,
		}
		idx++
	}
	// NB: PreCommits cannot be modified, but check anyway.
	for _, change := range preCommitChanges.Modified {
		out[idx] = &PreCommitChange{
			PreCommit: *change.Current,
			Change:    core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil
}
