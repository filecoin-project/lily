package verifreg

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type ClientExtractor struct{}

func (ClientExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "ClientExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "VerifiedRegistryClientExtractor.Transform")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewVerifiedRegistryExtractorContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("diffing verified registry clients: %w", err)
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
