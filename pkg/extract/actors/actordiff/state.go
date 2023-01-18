package actordiff

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiff struct {
	DiffMethods []actors.ActorStateDiff
}

func (s *StateDiff) State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorDiffResult, error) {
	start := time.Now()
	var stateDiff = new(StateDiffResult)
	for _, f := range s.DiffMethods {
		stateChange, err := f.Diff(ctx, api, act)
		if err != nil {
			return nil, err
		}
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindActorChange:
			stateDiff.ActorStateChanges = stateChange.(*ActorChange)
		}
	}
	log.Infow("Extracted Raw Actor State Diff", "address", act.Address, "duration", time.Since(start))
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
