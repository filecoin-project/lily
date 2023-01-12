package v0

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiffResult struct {
	AddressesChanges AddressChangeList
}

func (s *StateDiffResult) Kind() string {
	return "init"
}

func (s *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	out := &StateChange{}
	adtStore := store.WrapBlockStore(ctx, bs)

	if addresses := s.AddressesChanges; addresses != nil {
		root, err := addresses.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Addresses = root
	}
	return out, nil
}

type StateChange struct {
	Addresses cid.Cid `cborgen:"addresses"`
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
		case KindInitAddresses:
			stateDiff.AddressesChanges = stateChange.(AddressChangeList)
		}
	}
	log.Infow("Extracted Init State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
