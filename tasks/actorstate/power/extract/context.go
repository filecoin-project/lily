package extract

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/chain/actors/adt"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/power"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
	"golang.org/x/xerrors"
)

func NewPowerStateExtractionContext(ctx context.Context, a actor.ActorInfo, node actor.ActorStateAPI) (*PowerStateExtractionContext, error) {
	curState, err := power.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current power state: %w", err)
	}

	prevState := curState
	if a.Epoch != 1 {
		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &PowerStateExtractionContext{
					PrevState: prevState,
					CurrState: curState,
					CurrTs:    a.TipSet,
					Store:     node.Store(),
				}, nil
			}
			return nil, xerrors.Errorf("loading previous power actor at tipset %s epoch %d: %w", a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = power.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous power actor state: %w", err)
		}
	}
	return &PowerStateExtractionContext{
		PrevState: prevState,
		CurrState: curState,
		CurrTs:    a.TipSet,
		Store:     node.Store(),
	}, nil
}

type PowerStateExtractionContext struct {
	PrevState power.State
	CurrState power.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func (p *PowerStateExtractionContext) HasPreviousState() bool {
	return !(p.CurrTs.Height() == 1 || p.PrevState == p.CurrState)
}
