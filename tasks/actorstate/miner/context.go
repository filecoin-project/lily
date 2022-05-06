package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lotus/chain/types"

	miner "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

func NewMinerStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MinerStateExtractionContext, error) {
	ctx, span := otel.Tracer("").Start(ctx, "NewMinerExtractionContext")
	defer span.End()

	curState, err := miner.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current miner state: %w", err)
	}

	prevTipset := a.Current
	prevState := curState
	if a.Current.Height() != 1 {
		prevTipset = a.Executed

		prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &MinerStateExtractionContext{
					PrevState: prevState,
					PrevTs:    prevTipset,
					CurrActor: &a.Actor,
					CurrState: curState,
					CurrTs:    a.Current,
				}, nil
			}
			return nil, fmt.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.Executed.Key(), a.Current.Height(), err)
		}

		prevState, err = miner.Load(node.Store(), prevActor)
		if err != nil {
			return nil, fmt.Errorf("loading previous miner actor state: %w", err)
		}
	}

	return &MinerStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.Current,
	}, nil
}

type MinerStateExtractionContext struct {
	PrevState miner.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState miner.State
	CurrTs    *types.TipSet
}

func (m *MinerStateExtractionContext) HasPreviousState() bool {
	return !(m.CurrTs.Height() == 1 || m.PrevState == m.CurrState)
}
