package market

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/runes"

	"github.com/filecoin-project/lily/chain/actors/adt"
	market "github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lotus/chain/types"
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

// SanitizeLabel ensures:
// - s is a valid utf8 string by removing any ill formed bytes.
// - s does not contain any nil (\x00) bytes because postgres doesn't support storing NULL (\0x00) characters in text fields.
func SanitizeLabel(s string) string {
	if s == "" {
		return s
	}
	s = strings.Replace(s, "\000", "", -1)
	if utf8.ValidString(s) {
		return s
	}

	tr := runes.ReplaceIllFormed()
	return tr.String(s)
}
