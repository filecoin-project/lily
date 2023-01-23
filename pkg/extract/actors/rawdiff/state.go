package rawdiff

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiff struct {
	DiffMethods []actors.ActorStateDiff
}

func (s *StateDiff) State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorDiffResult, error) {
	grp, grpctx := errgroup.WithContext(ctx)
	results, err := actors.ExecuteStateDiff(grpctx, grp, api, act, s.DiffMethods...)
	if err != nil {
		return nil, err
	}
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range results {
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindActorChange:
			stateDiff.ActorStateChanges = stateChange.(*ActorChange)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	ActorStateChanges *ActorChange
}

func (sdr *StateDiffResult) Kind() string {
	return "actor"
}

func (sdr *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	return sdr.ActorStateChanges, nil
}

type StateChange struct {
	ActorState cid.Cid `cborgen:"actors"`
}

func GenericStateDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, diffFns []actors.ActorStateDiff) ([]actors.ActorStateChange, error) {
	grp, grpCtx := errgroup.WithContext(ctx)
	out := make([]actors.ActorStateChange, 0, len(diffFns))
	results := make(chan actors.ActorStateChange, len(diffFns))
	for _, diff := range diffFns {
		diff := diff
		grp.Go(func() error {
			res, err := diff.Diff(grpCtx, api, act)
			if err != nil {
				return err
			}
			results <- res
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		close(results)
		return nil, err
	}
	close(results)
	for res := range results {
		out = append(out, res)
	}
	return out, nil
}
