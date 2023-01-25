package v9

import (
	"context"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v8"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/verifreg")

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
		case v0.KindVerifregVerifiers:
			stateDiff.VerifierChanges = stateChange.(VerifiersChangeList)
		case KindVerifregClaims:
			stateDiff.ClaimChanges = stateChange.(ClaimsChangeMap)
		case KindVerifregAllocations:
			stateDiff.AllocationsChanges = stateChange.(AllocationsChangeMap)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	VerifierChanges    v8.VerifiersChangeList
	ClaimChanges       ClaimsChangeMap
	AllocationsChanges AllocationsChangeMap
}

func (sd *StateDiffResult) Kind() string {
	return "verifreg"
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
