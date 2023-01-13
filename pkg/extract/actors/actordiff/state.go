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
	return s.ActorStateChanges, nil
	out := &StateChange{}

	if actorChanges := s.ActorStateChanges; actorChanges != nil {
		blk, err := actorChanges.ToStorageBlock()
		if err != nil {
			return nil, err
		}
		if err := bs.Put(ctx, blk); err != nil {
			return nil, err
		}
		c := blk.Cid()
		out.ActorState = c
	} else {
		return nil, nil
	}
	return out, nil
}

func (s *StateDiffResult) Kind() string {
	return "actor"
}

type StateChange struct {
	ActorState cid.Cid `cborgen:"actors"`
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
	log.Infow("Extracted Raw Actor State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
