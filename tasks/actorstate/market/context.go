package market

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/tasks/actorstate"

	market "github.com/filecoin-project/lily/chain/actors/builtin/market"
)

type MarketStateExtractionContext struct {
	PrevState market.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState market.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func NewMarketStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MarketStateExtractionContext, error) {
	curState, err := market.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current market state: %w", err)
	}

	prevTipset := a.Current
	prevState := curState
	if a.Current.Height() != 0 {
		prevTipset = a.Executed

		prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
		if err != nil {
			return nil, fmt.Errorf("loading previous market actor state at tipset %s epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)
		}

		prevState, err = market.Load(node.Store(), prevActor)
		if err != nil {
			return nil, fmt.Errorf("loading previous market actor state: %w", err)
		}

	}
	return &MarketStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.Current,
		Store:     node.Store(),
	}, nil
}

func (m *MarketStateExtractionContext) IsGenesis() bool {
	return m.CurrTs.Height() == 0
}
