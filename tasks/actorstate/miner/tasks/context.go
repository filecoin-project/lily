package tasks

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/chain/actors/adt"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"
)

func NewMinerStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MinerStateExtractionContext, error) {
	ctx, span := global.Tracer("").Start(ctx, "NewMinerExtractionContext")
	defer span.End()

	curState, err := miner.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner state: %w", err)
	}

	prevTipset := a.TipSet
	prevState := curState
	if a.Epoch != 1 {
		prevTipset = a.ParentTipSet

		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &MinerStateExtractionContext{
					PrevState: prevState,
					PrevTs:    prevTipset,
					CurrActor: &a.Actor,
					CurrState: curState,
					CurrTs:    a.TipSet,
				}, nil
			}
			return nil, xerrors.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = miner.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous miner actor state: %w", err)
		}
	}

	return &MinerStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.TipSet,
		Address:   a.Address,
		Store:     node.Store(),
		API:       node,
		Cache:     NewDiffCache(),
	}, nil
}

type MinerStateExtractionContext struct {
	PrevState miner.State
	PrevTs    *types.TipSet

	CurrState miner.State
	CurrTs    *types.TipSet

	CurrActor *types.Actor
	Address   address.Address

	Store adt.Store
	API   actorstate.ActorStateAPI

	Cache *DiffCache
}

func (m *MinerStateExtractionContext) HasPreviousState() bool {
	return !(m.CurrTs.Height() == 1 || m.PrevState == m.CurrState)
}
