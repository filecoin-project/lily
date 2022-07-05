package verifreg

import (
	"context"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/tasks/actorstate"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/model"
)

type VerifiedRegistryExtractor struct{}

type VerifiedRegistryExtractionContext struct {
	PrevState, CurrState verifreg.State
	PrevTs, CurrTs       *types.TipSet

	Store                adt.Store
	PreviousStatePresent bool
}

func (v *VerifiedRegistryExtractionContext) HasPreviousState() bool {
	return v.PreviousStatePresent
}

func NewVerifiedRegistryExtractorContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*VerifiedRegistryExtractionContext, error) {
	curState, err := verifreg.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current verified registry state: %w", err)
	}

	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// actor doesn't exist yet, may have just been created.
		if err == types.ErrActorNotFound {
			return &VerifiedRegistryExtractionContext{
				CurrState:            curState,
				PrevTs:               a.Executed,
				CurrTs:               a.Current,
				Store:                node.Store(),
				PrevState:            nil,
				PreviousStatePresent: false,
			}, nil
		}
		return nil, fmt.Errorf("loading previous verified registry actor from parent tipset %s current height epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)
	}

	// actor exists in previous state, load it.
	prevState, err := verifreg.Load(node.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous verified registry state: %w", err)
	}
	return &VerifiedRegistryExtractionContext{
		PrevState:            prevState,
		CurrState:            curState,
		PrevTs:               a.Executed,
		CurrTs:               a.Current,
		Store:                node.Store(),
		PreviousStatePresent: true,
	}, nil
}

func (VerifiedRegistryExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "VerifiedRegistryExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	verifiers, err := VerifierExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	clients, err := ClientExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, err
	}

	return model.PersistableList{
		verifiers,
		clients,
	}, nil
}
