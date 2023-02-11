package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.DiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range changes {
		switch v := stateChange.(type) {
		case DealChangeList:
			stateDiff.DealStateChanges = v
		case ProposalChangeList:
			stateDiff.DealProposalChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	DealStateChanges    DealChangeList
	DealProposalChanges ProposalChangeList
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	out := &StateChange{}

	if deals := sd.DealStateChanges; deals != nil {
		root, err := deals.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Deals = &root
	}

	if proposals := sd.DealProposalChanges; proposals != nil {
		root, err := proposals.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Proposals = &root
	}
	return out, nil
}

type StateChange struct {
	Deals     *cid.Cid `cborgen:"deals"`
	Proposals *cid.Cid `cborgen:"proposals"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{
		DealStateChanges:    DealChangeList{},
		DealProposalChanges: ProposalChangeList{},
	}

	if sc.Deals != nil {
		dealsMap, err := adt.AsMap(s, *sc.Deals, 5)
		if err != nil {
			return nil, err
		}
		deals := DealChangeList{}
		dealChange := new(DealChange)
		if err := dealsMap.ForEach(dealChange, func(key string) error {
			val := new(DealChange)
			*val = *dealChange
			// NB: this is optinal since the dealChange already contains the dealID
			// TODO consider removeing the key from the structure to save space
			dealID, err := abi.ParseUIntKey(key)
			if err != nil {
				return err
			}
			if dealID != val.DealID {
				panic("here")
			}
			val.DealID = dealID
			deals = append(deals, val)
			return nil
		}); err != nil {
			return nil, err
		}
		out.DealStateChanges = deals
	}

	if sc.Proposals != nil {
		propsMap, err := adt.AsMap(s, *sc.Proposals, 5)
		if err != nil {
			return nil, err
		}
		props := ProposalChangeList{}
		propChange := new(ProposalChange)
		if err := propsMap.ForEach(propChange, func(key string) error {
			val := new(ProposalChange)
			*val = *propChange
			// NB: this is optinal since the dealChange already contains the dealID
			// TODO consider removeing the key from the structure to save space
			dealID, err := abi.ParseUIntKey(key)
			if err != nil {
				return err
			}
			if dealID != val.DealID {
				panic("here")
			}
			val.DealID = dealID
			props = append(props, val)
			return nil
		}); err != nil {
			return nil, err
		}
		out.DealProposalChanges = props
	}
	return out, nil
}
