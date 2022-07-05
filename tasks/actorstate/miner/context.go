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

	curState, err := node.MinerLoad(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current miner state: %w", err)
	}

	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// actor doesn't exist yet, may have just been created.
		if err == types.ErrActorNotFound {
			return &MinerStateExtractionContext{
				PrevTs:               a.Executed,
				CurrActor:            &a.Actor,
				CurrState:            curState,
				CurrTs:               a.Current,
				PrevState:            nil,
				PreviousStatePresent: false,
			}, nil
		}
		return nil, fmt.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.Executed.Key(), a.Current.Height(), err)
	}

	// actor exists in previous state, load it.
	prevState, err := node.MinerLoad(node.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous miner actor state: %w", err)
	}

	return &MinerStateExtractionContext{
		PrevState:            prevState,
		PrevTs:               a.Executed,
		CurrActor:            &a.Actor,
		CurrState:            curState,
		CurrTs:               a.Current,
		PreviousStatePresent: true,
	}, nil
}

type MinerStateExtractionContext struct {
	PrevState miner.State
	PrevTs    *types.TipSet

	CurrActor            *types.Actor
	CurrState            miner.State
	CurrTs               *types.TipSet
	PreviousStatePresent bool
}

func (m *MinerStateExtractionContext) HasPreviousState() bool {
	return m.PreviousStatePresent
}
