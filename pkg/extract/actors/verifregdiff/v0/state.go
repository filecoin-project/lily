package v0

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiffResult struct {
	VerifierChanges VerifiersChangeList
	ClientChanges   ClientsChangeList
}

func (s *StateDiffResult) Kind() string {
	return "verifreg"
}

type StateChange struct {
	Verifiers cid.Cid
	Clients   cid.Cid
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (typegen.CBORMarshaler, error) {
	panic("TODO")
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
		case KindVerifregClients:
			stateDiff.ClientChanges = stateChange.(ClientsChangeList)
		case KindVerifregVerifiers:
			stateDiff.VerifierChanges = stateChange.(VerifiersChangeList)
		}
	}
	log.Infow("Extracted Verifid Registry State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
