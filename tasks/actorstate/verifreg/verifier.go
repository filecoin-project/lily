package verifreg

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/verifreg")

type VerifierExtractor struct{}

func (VerifierExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "VerifierExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "VerifierExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewVerifiedRegistryExtractorContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("diffing verified registry verifiers: %w", err)
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
