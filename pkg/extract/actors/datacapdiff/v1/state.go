package v1

import (
	"context"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

type StateDiffResult struct {
	BalanceChanges   BalanceChangeList
	AllowanceChanges AllowanceChangeMap
}

func (sd *StateDiffResult) Kind() string {
	return "datacap"
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

type StateDiff struct {
	DiffMethods []actors.ActorStateDiff
}

func (s *StateDiff) State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorDiffResult, error) {
	return ChangeHandler(
		ctx,
		api,
		act,
		func(changes []actors.ActorStateChange) (actors.ActorDiffResult, error) {
			var stateDiff = new(StateDiffResult)
			for _, stateChange := range changes {
				switch stateChange.Kind() {
				case KindDataCapAllowance:
					stateDiff.AllowanceChanges = stateChange.(AllowanceChangeMap)
				case KindDataCapBalance:
					stateDiff.BalanceChanges = stateChange.(BalanceChangeList)
				}
			}
			return stateDiff, nil
		},
		s.DiffMethods...)
}

type StateDiffHandlerFn = func(changes []actors.ActorStateChange) (actors.ActorDiffResult, error)

func ChangeHandler(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, handler StateDiffHandlerFn, differs ...actors.ActorStateDiff) (actors.ActorDiffResult, error) {
	grp, grpctx := errgroup.WithContext(ctx)
	results, err := actors.ExecuteStateDiff(grpctx, grp, api, act, differs...)
	if err != nil {
		return nil, err
	}
	return handler(results)
}
