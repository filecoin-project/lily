package rawdiff

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.ActorDiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, change := range changes {
		switch change.Kind() {
		case KindActorChange:
			stateDiff.ActorStateChanges = change.(*ActorChange)
		default:
			return nil, fmt.Errorf("unhandledd change kind: %s", change.Kind())
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	ActorStateChanges *ActorChange
}

func (sdr *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	return sdr.ActorStateChanges, nil
}

type StateChange struct {
	ActorState cid.Cid `cborgen:"actors"`
}
