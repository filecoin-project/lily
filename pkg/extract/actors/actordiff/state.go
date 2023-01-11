package actordiff

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiffResult struct {
	ActorStateChanges *ActorChange
}

func (s *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StateDiffResult) Kind() string {
	return "actor"
}

type StateChange struct {
	ActorStates cid.Cid // Hamt[address]StateChange
}

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
	log.Infow("Extracted Init State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
