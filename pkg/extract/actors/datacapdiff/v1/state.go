package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors"
)

func ActorStateChangeHandler(changes []actors.ActorStateChange) (actors.DiffResult, error) {
	var stateDiff = new(StateDiffResult)
	for _, stateChange := range changes {
		switch v := stateChange.(type) {
		case AllowanceChangeMap:
			stateDiff.AllowanceChanges = v
		case BalanceChangeList:
			stateDiff.BalanceChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	BalanceChanges   BalanceChangeList
	AllowanceChanges AllowanceChangeMap
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	out := &StateChange{}

	if balances := sd.BalanceChanges; balances != nil {
		root, err := balances.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Balances = &root
	}
	if allowances := sd.AllowanceChanges; allowances != nil {
		root, err := allowances.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Allowances = &root
	}
	return out, nil
}

type StateChange struct {
	Balances   *cid.Cid `cborgen:"balances"`
	Allowances *cid.Cid `cborgen:"allowances"`
}
