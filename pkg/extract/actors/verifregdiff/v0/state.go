package v0

import (
	"context"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
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
		case KindVerifregClients:
			stateDiff.ClientChanges = stateChange.(ClientsChangeList)
		case KindVerifregVerifiers:
			stateDiff.VerifierChanges = stateChange.(VerifiersChangeList)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	VerifierChanges VerifiersChangeList
	ClientChanges   ClientsChangeList
}

func (s *StateDiffResult) Kind() string {
	return "verifreg"
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
