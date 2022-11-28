package minerdiff

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

var _ ActorStateChange = (*PreCommitChangeList)(nil)

type PreCommitChange struct {
	PreCommit typegen.Deferred
	Type      core.ChangeType
}

type PreCommitChangeList []*PreCommitChange

func (p PreCommitChangeList) Kind() ActorStateKind {
	return "miner_precommit"
}

type PreCommit struct{}

func (PreCommit) Diff(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error) {
	return DiffPreCommits(ctx, api, act, current, executed)
}

func DiffPreCommits(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error) {
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
		var out PreCommitChangeList
		pm, err := currentMiner.PrecommitsMap()
		if err != nil {
			return nil, err
		}
		var v typegen.Deferred
		if err := pm.ForEach(&v, func(key string) error {
			out = append(out, &PreCommitChange{
				PreCommit: v,
				Type:      core.ChangeTypeAdd,
			})
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

	preCommitChanges, err := miner.DiffPreCommitsDeferred(ctx, api.Store(), executedMiner, currentMiner)
	if err != nil {
		return nil, err
	}

	idx := 0
	out := make(PreCommitChangeList, len(preCommitChanges.Added)+len(preCommitChanges.Removed)+len(preCommitChanges.Modified))
	for _, change := range preCommitChanges.Added {
		out[idx] = &PreCommitChange{
			PreCommit: *change,
			Type:      core.ChangeTypeAdd,
		}
		idx++
	}
	for _, change := range preCommitChanges.Removed {
		out[idx] = &PreCommitChange{
			PreCommit: *change,
			Type:      core.ChangeTypeRemove,
		}
		idx++
	}
	// NB: PreCommits cannot be modified, but check anyway.
	for _, change := range preCommitChanges.Modified {
		out[idx] = &PreCommitChange{
			PreCommit: *change.Current,
			Type:      core.ChangeTypeModify,
		}
		idx++
	}
	return out, nil
}
