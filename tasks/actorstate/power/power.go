package power

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/model"
)

// was services/processor/tasks/power/power.go

// StoragePowerExtractor extracts power actor state
type StoragePowerExtractor struct{}

func NewPowerStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*PowerStateExtractionContext, error) {
	curState, err := power.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current power state: %w", err)
	}

	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// actor doesn't exist yet, may have just been created.
		if err == types.ErrActorNotFound {
			return &PowerStateExtractionContext{
				CurrState:            curState,
				CurrTs:               a.Current,
				Store:                node.Store(),
				PrevState:            nil,
				PreviousStatePresent: false,
			}, nil
		}
		return nil, fmt.Errorf("loading previous power actor from parent tipset %s current epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)
	}

	// actor exists in previous state, load it.
	prevState, err := power.Load(node.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous power actor state: %w", err)
	}
	return &PowerStateExtractionContext{
		PrevState:            prevState,
		CurrState:            curState,
		CurrTs:               a.Current,
		Store:                node.Store(),
		PreviousStatePresent: true,
	}, nil
}

type PowerStateExtractionContext struct {
	PrevState power.State
	CurrState power.State
	CurrTs    *types.TipSet

	Store                adt.Store
	PreviousStatePresent bool
}

func (p *PowerStateExtractionContext) HasPreviousState() bool {
	return p.PreviousStatePresent
}

func (StoragePowerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StoragePowerExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

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
