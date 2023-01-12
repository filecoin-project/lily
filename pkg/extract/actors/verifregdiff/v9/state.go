package v9

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v8"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/verifreg")

type StateDiffResult struct {
	VerifierChanges    v8.VerifiersChangeList
	ClaimChanges       ClaimsChangeMap
	AllocationsChanges AllocationsChangeMap
}

func (sd *StateDiffResult) Kind() string {
	return "verifreg"
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	out := &StateChange{}
	adtStore := store.WrapBlockStore(ctx, bs)

	if verifiers := sd.VerifierChanges; verifiers != nil {
		root, err := verifiers.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Verifiers = &root
	}

	if claims := sd.ClaimChanges; claims != nil {
		root, err := claims.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Claims = &root
	}

	if allocations := sd.AllocationsChanges; allocations != nil {
		root, err := allocations.ToAdtMap(adtStore, 5)
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
		case v0.KindVerifregVerifiers:
			stateDiff.VerifierChanges = stateChange.(VerifiersChangeList)
		case KindVerifregClaims:
			stateDiff.ClaimChanges = stateChange.(ClaimsChangeMap)
		case KindVerifregAllocations:
			stateDiff.AllocationsChanges = stateChange.(AllocationsChangeMap)
		}
	}
	log.Infow("Extracted Verified Registry State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
