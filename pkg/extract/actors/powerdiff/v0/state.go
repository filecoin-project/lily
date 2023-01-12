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
	ClaimsChanges ClaimsChangeList
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (cbg.CBORMarshaler, error) {
	out := &StateChange{}
	adtStore := store.WrapBlockStore(ctx, bs)

	if claims := sd.ClaimsChanges; claims != nil {
		root, err := claims.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Claims = root
	}

	return out, nil
}

func (sd *StateDiffResult) Kind() string {
	return "power"
}

type StateChange struct {
	Claims cid.Cid `cborgen:"claims"`
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
		case KindPowerClaims:
			stateDiff.ClaimsChanges = stateChange.(ClaimsChangeList)
		default:
			panic(stateChange.Kind())
		}
	}
	log.Infow("Extracted Power State Diff", "address", act.Address, "duration", time.Since(start))
	return stateDiff, nil
}
