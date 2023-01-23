package v0

import (
	"context"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
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
		case KindInitAddresses:
			stateDiff.AddressesChanges = stateChange.(AddressChangeList)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	AddressesChanges AddressChangeList
}

func (s *StateDiffResult) Kind() string {
	return "init"
}

func (sdr *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	out := &StateChange{}

	if addresses := sdr.AddressesChanges; addresses != nil {
		root, err := addresses.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Addresses = &root
	}
	return out, nil
}

type StateChange struct {
	Addresses *cid.Cid `cborgen:"addresses"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{AddressesChanges: AddressChangeList{}}

	if sc.Addresses != nil {
		addressMap, err := adt.AsMap(s, *sc.Addresses, 5)
		if err != nil {
			return nil, err
		}

		addresses := AddressChangeList{}
		addressChange := new(AddressChange)
		if err := addressMap.ForEach(addressChange, func(key string) error {
			val := new(AddressChange)
			*val = *addressChange
			// NB: not required
			// TODO consider removing they key from the structure.
			val.Address = []byte(key)
			addresses = append(addresses, val)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}
