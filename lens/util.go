package lens

import (
	"context"
	"errors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/impl/full"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var logger = logging.Logger("visor/lens/lotus")

// OptimizedStateGetActorWithFallback is a helper to obtain an actor in the
// state of the current tipset without recomputing the full tipset. It does
// this by obtaining the child tipset (current height+1) and using the
// pre-computed ParentState().
//
// TODO: Remove. See:  https://github.com/filecoin-project/sentinel-visor/issues/196
func OptimizedStateGetActorWithFallback(ctx context.Context, store *store.ChainStore, fallback full.StateModuleAPI, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	act, err := efficientStateGetActorFromChainStore(ctx, store, actor, tsk)
	if err != nil {
		logger.Warnf("Optimized StateGetActorError: %s. Falling back to default StateGetActor().", err)
		return fallback.StateGetActor(ctx, actor, tsk)
	}
	return act, nil
}

func efficientStateGetActorFromChainStore(ctx context.Context, store *store.ChainStore, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	ts, err := store.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("Failed to load tipset: %w", err)
	}

	// heaviest tipset means look on the main chain and false means return tipset following null round.
	child, err := store.GetTipsetByHeight(ctx, ts.Height()+1, store.GetHeaviestTipSet(), false)
	if err != nil {
		return nil, xerrors.Errorf("load child tipset: %w", err)
	}

	if !types.CidArrsEqual(child.Parents().Cids(), ts.Cids()) {
		return nil, errors.New("child is not on the same chain")
	}

	st, err := state.LoadStateTree(store.Store(ctx), child.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}
	return st.GetActor(actor)
}
