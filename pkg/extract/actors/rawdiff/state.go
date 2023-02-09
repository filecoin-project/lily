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
		switch v := change.(type) {
		case *ActorChange:
			stateDiff.ActorStateChanges = v
		default:
			return nil, fmt.Errorf("unhandled change type: %T", v)
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
