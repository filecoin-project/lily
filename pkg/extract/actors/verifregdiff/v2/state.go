package v2

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

var log = logging.Logger("extract/actors/verifreg")

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.ActorDiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range changes {
		switch v := stateChange.(type) {
		case VerifiersChangeList:
			stateDiff.VerifierChanges = v
		case ClaimsChangeMap:
			stateDiff.ClaimChanges = v
		case AllocationsChangeMap:
			stateDiff.AllocationsChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil

}

type StateDiffResult struct {
	VerifierChanges    VerifiersChangeList
	ClaimChanges       ClaimsChangeMap
	AllocationsChanges AllocationsChangeMap
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, store store.Store) (cbg.CBORMarshaler, error) {
	out := &StateChange{}

	if verifiers := sd.VerifierChanges; verifiers != nil {
		root, err := verifiers.ToAdtMap(store, 5)
		if err != nil {
			return nil, err
		}
		out.Verifiers = &root
	}

	if claims := sd.ClaimChanges; claims != nil {
		root, err := claims.ToAdtMap(store, 5)
		if err != nil {
			return nil, err
		}
		out.Claims = &root
	}

	if allocations := sd.AllocationsChanges; allocations != nil {
		root, err := allocations.ToAdtMap(store, 5)
		if err != nil {
			return nil, err
		}
		out.Allocations = &root
	}
	return out, nil
}

type StateChange struct {
	Verifiers   *cid.Cid `cborgen:"verifiers"`
	Claims      *cid.Cid `cborgen:"claims"`
	Allocations *cid.Cid `cborgen:"allocations"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{
		VerifierChanges: VerifiersChangeList{},
	}

	if sc.Verifiers != nil {
		verifierMap, err := adt.AsMap(s, *sc.Verifiers, 5)
		if err != nil {
			return nil, err
		}

		verifierChange := new(VerifiersChange)
		if err := verifierMap.ForEach(verifierChange, func(key string) error {
			val := new(VerifiersChange)
			*val = *verifierChange
			out.VerifierChanges = append(out.VerifierChanges, val)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return out, nil
}
