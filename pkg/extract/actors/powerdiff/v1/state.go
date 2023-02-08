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
		case ClaimsChangeList:
			stateDiff.ClaimsChanges = v
		default:
			return nil, fmt.Errorf("unknown state change kind: %T", v)
		}
	}
	return stateDiff, nil
}

type StateDiffResult struct {
	ClaimsChanges ClaimsChangeList
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (cbg.CBORMarshaler, error) {
	out := &StateChange{}
	if claims := sd.ClaimsChanges; claims != nil {
		root, err := claims.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Claims = &root
	}
	return out, nil
}

type StateChange struct {
	Claims *cid.Cid `cborgen:"claims"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{ClaimsChanges: ClaimsChangeList{}}

	if sc.Claims != nil {
		claimMap, err := adt.AsMap(s, *sc.Claims, 5)
		if err != nil {
			return nil, err
		}

		claims := ClaimsChangeList{}
		claimChange := new(ClaimsChange)
		if err := claimMap.ForEach(claimChange, func(key string) error {
			val := new(ClaimsChange)
			*val = *claimChange
			claims = append(claims, val)
			return nil
		}); err != nil {
			return nil, err
		}
		out.ClaimsChanges = claims
	}
	return out, nil
}
