package power

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// was services/processor/tasks/power/power.go

// StoragePowerExtractor extracts power actor state
type StoragePowerExtractor struct{}

func NewPowerStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*PowerStateExtractionContext, error) {
	curState, err := power.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current power state: %w", err)
	}

	prevState := curState
	if a.Current.Height() != 1 {
		prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &PowerStateExtractionContext{
					PrevState: prevState,
					CurrState: curState,
					CurrTs:    a.Current,
					Store:     node.Store(),
				}, nil
			}
			return nil, xerrors.Errorf("loading previous power actor at tipset %s epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)
		}

		prevState, err = power.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous power actor state: %w", err)
		}
	}
	return &PowerStateExtractionContext{
		PrevState: prevState,
		CurrState: curState,
		CurrTs:    a.Current,
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

func (StoragePowerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StoragePowerExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	chainPowerModel, err := ChainPowerExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	claimedPowerModel, err := ClaimedPowerExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}
	return model.PersistableList{
		chainPowerModel,
		claimedPowerModel,
	}, nil
}
