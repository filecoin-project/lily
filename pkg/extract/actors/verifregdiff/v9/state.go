package v9

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/blockstore"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v0"
	v8 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v8"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/verifreg")

type StateDiffResult struct {
	VerifierChanges    v8.VerifiersChangeList
	ClaimChanges       ClaimsChangeList
	AllocationsChanges AllocationsChangeList
}

func (s *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StateDiffResult) Kind() string {
	return "verifreg"
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
			stateDiff.VerifierChanges = stateChange.(v0.VerifiersChangeList)
		case KindVerifregClaims:
			stateDiff.ClaimChanges = stateChange.(ClaimsChangeList)
		case KindVerifregAllocations:
			stateDiff.AllocationsChanges = stateChange.(AllocationsChangeList)
		}
	}
	log.Infow("Extracted Verifid Registry State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
