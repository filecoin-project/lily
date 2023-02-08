package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.ActorDiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range changes {
		switch v := stateChange.(type) {
		case AddressChangeList:
			stateDiff.AddressesChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	AddressesChanges AddressChangeList
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
