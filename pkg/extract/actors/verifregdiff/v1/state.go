package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.ActorDiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range changes {
		switch v := stateChange.(type) {
		case ClientsChangeList:
			stateDiff.ClientChanges = v
		case VerifiersChangeList:
			stateDiff.VerifierChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	VerifierChanges VerifiersChangeList
	ClientChanges   ClientsChangeList
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (typegen.CBORMarshaler, error) {
	out := &StateChange{}

	if clients := sd.ClientChanges; clients != nil {
		root, err := clients.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Clients = &root
	}

	if verifiers := sd.VerifierChanges; verifiers != nil {
		root, err := verifiers.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Verifiers = &root
	}
	return out, nil
}

type StateChange struct {
	Verifiers *cid.Cid `cborgen:"verifiers"`
	Clients   *cid.Cid `cborgen:"clients"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{
		VerifierChanges: VerifiersChangeList{},
		ClientChanges:   ClientsChangeList{},
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

	if sc.Clients != nil {
		clientsMap, err := adt.AsMap(s, *sc.Clients, 5)
		if err != nil {
			return nil, err
		}

		clientChange := new(ClientsChange)
		if err := clientsMap.ForEach(clientChange, func(key string) error {
			val := new(ClientsChange)
			*val = *clientChange
			out.ClientChanges = append(out.ClientChanges, val)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}
