package actorstate

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"golang.org/x/xerrors"

	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type VerifiedRegistryExtractor struct{}

func init() {
	for _, c := range verifreg.AllCodes() {
		Register(c, VerifiedRegistryExtractor{})
	}
}

type VerifiedRegistryExtractionContext struct {
	PrevState, CurrState verifreg.State
	PrevTs, CurrTs       *types.TipSet

	Store adt.Store
}

func (v *VerifiedRegistryExtractionContext) HasPreviousState() bool {
	return !(v.CurrTs.Height() == 1 || v.PrevState == v.CurrState)
}

func NewVerifiedRegistryExtractorContext(ctx context.Context, a ActorInfo, node ActorStateAPI) (*VerifiedRegistryExtractionContext, error) {
	curState, err := verifreg.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current verified registry state: %w", err)
	}

	prevState := curState
	if a.Epoch != 0 {
		prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet.Key())
		if err != nil {
			// if the actor exists in the current state and not in the parent state then the
			// actor was created in the current state.
			if err == types.ErrActorNotFound {
				return &VerifiedRegistryExtractionContext{
					PrevState: prevState,
					CurrState: curState,
					PrevTs:    a.ParentTipSet,
					CurrTs:    a.TipSet,
					Store:     node.Store(),
				}, nil
			}
			return nil, xerrors.Errorf("loading previous verified registry actor at tipset %s epoch %d: %w", a.ParentTipSet.Key(), a.Epoch, err)
		}

		prevState, err = verifreg.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous verified registry state: %w", err)
		}
	}
	return &VerifiedRegistryExtractionContext{
		PrevState: prevState,
		CurrState: curState,
		PrevTs:    a.ParentTipSet,
		CurrTs:    a.TipSet,
		Store:     node.Store(),
	}, nil
}

func (VerifiedRegistryExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "VerifiedRegistryExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	ec, err := NewVerifiedRegistryExtractorContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	verifiers, err := ExtractVerifiers(ctx, ec)
	if err != nil {
		return nil, err
	}

	clients, err := ExtractVerifiedClients(ctx, ec)
	if err != nil {
		return nil, err
	}

	return model.PersistableList{
		verifiers,
		clients,
	}, nil
}

func ExtractVerifiers(ctx context.Context, ec *VerifiedRegistryExtractionContext) (verifregmodel.VerifiedRegistryVerifiersList, error) {
	var verifiers verifregmodel.VerifiedRegistryVerifiersList
	// if this is the genesis state extract whatever state it has, there is noting to diff against
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachVerifier(func(addr address.Address, dcap abi.StoragePower) error {
			verifiers = append(verifiers, &verifregmodel.VerifiedRegistryVerifier{
				Height:    int64(ec.CurrTs.Height()),
				StateRoot: ec.CurrTs.ParentState().String(),
				Address:   addr.String(),
				DataCap:   dcap.String(),
				Event:     verifregmodel.Added,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return verifiers, nil
	}

	changes, err := verifreg.DiffVerifiers(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, xerrors.Errorf("diffing verified registry verifiers: %w", err)
	}

	// a new verifier was added
	for _, change := range changes.Added {
		verifiers = append(verifiers, &verifregmodel.VerifiedRegistryVerifier{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.Address.String(),

			DataCap: change.DataCap.String(),
			Event:   verifregmodel.Added,
		})
	}
	// a verifier was removed
	for _, change := range changes.Removed {
		verifiers = append(verifiers, &verifregmodel.VerifiedRegistryVerifier{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.Address.String(),

			DataCap: "0",
			Event:   verifregmodel.Removed,
		})
	}
	// an existing verifier's DataCap changed
	for _, change := range changes.Modified {
		verifiers = append(verifiers, &verifregmodel.VerifiedRegistryVerifier{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.After.Address.String(),

			DataCap: change.After.DataCap.String(),
			Event:   verifregmodel.Modified,
		})
	}
	return verifiers, nil
}

func ExtractVerifiedClients(ctx context.Context, ec *VerifiedRegistryExtractionContext) (verifregmodel.VerifiedRegistryVerifiedClientsList, error) {
	var clients verifregmodel.VerifiedRegistryVerifiedClientsList
	// if this is the genesis state extract whatever state it has, there is noting to diff against
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClient(func(addr address.Address, dcap abi.StoragePower) error {
			clients = append(clients, &verifregmodel.VerifiedRegistryVerifiedClient{
				Height:    int64(ec.CurrTs.Height()),
				StateRoot: ec.CurrTs.ParentState().String(),
				Address:   addr.String(),
				DataCap:   dcap.String(),
				Event:     verifregmodel.Added,
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return clients, nil
	}

	changes, err := verifreg.DiffVerifiedClients(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, xerrors.Errorf("diffing verified registry clients: %w", err)
	}

	for _, change := range changes.Added {
		clients = append(clients, &verifregmodel.VerifiedRegistryVerifiedClient{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.Address.String(),
			DataCap:   change.DataCap.String(),
			Event:     verifregmodel.Added,
		})
	}
	for _, change := range changes.Modified {
		clients = append(clients, &verifregmodel.VerifiedRegistryVerifiedClient{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.After.Address.String(),
			DataCap:   change.After.DataCap.String(),
			Event:     verifregmodel.Modified,
		})
	}
	for _, change := range changes.Removed {
		clients = append(clients, &verifregmodel.VerifiedRegistryVerifiedClient{
			Height:    int64(ec.CurrTs.Height()),
			StateRoot: ec.CurrTs.ParentState().String(),
			Address:   change.Address.String(),
			DataCap:   "0",
			Event:     verifregmodel.Removed,
		})
	}
	return clients, nil
}
