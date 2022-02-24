package miner

import (
	"context"

	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/types"

	miner "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/model"
)

// was services/processor/tasks/miner/miner.go

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func (m StorageMinerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StorageMinerExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	minerInfoModel, err := InfoExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner info: %w", err)
	}

	lockedFundsModel, err := LockedFundsExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner locked funds: %w", err)
	}

	feeDebtModel, err := FeeDebtExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner fee debt: %w", err)
	}

	currDeadlineModel, err := DeadlineInfoExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner current deadline info: %w", err)
	}

	preCommitModel, err := PreCommitInfoExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	sectorModel, err := SectorInfoExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	sectorDealsModel, err := SectorDealsExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	sectorEventsModel, err := SectorEventsExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	posts, err := PoStExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	return model.PersistableList{
		minerInfoModel,
		lockedFundsModel,
		feeDebtModel,
		currDeadlineModel,
		preCommitModel,
		sectorModel,
		sectorDealsModel,
		sectorEventsModel,
		posts,
	}, nil
}

func NewMinerStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MinerStateExtractionContext, error) {
	ctx, span := otel.Tracer("").Start(ctx, "NewMinerExtractionContext")
	defer span.End()

	curState, err := miner.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current miner state: %w", err)
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
			return nil, xerrors.Errorf("loading previous miner %s at tipset %s epoch %d: %w", a.Address, a.Executed.Key(), a.Current.Height(), err)
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
