package datacap

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/builtin/datacap"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/model"
	datacapmodel "github.com/filecoin-project/lily/model/actors/datacap"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks/datacap")

type BalanceExtractor struct{}

func (extractor BalanceExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "BalanceExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "BalancesExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewBalanceExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	var balances datacapmodel.DataCapBalanceList

	// should only be true on the upgrade boundary
	if !ec.HasPreviousState() {
		if err := ec.CurrState.ForEachClient(func(addr address.Address, dcap abi.StoragePower) error {
			balances = append(balances, &datacapmodel.DataCapBalance{
				Height:      int64(ec.CurrTs.Height()),
				StateRoot:   ec.CurrTs.ParentState().String(),
				Address:     addr.String(),
				Event:       datacapmodel.Added,
				DataCap:     dcap.String(),
				AddressType: datacapmodel.VerifierClient,
			})
			return nil
		}); err != nil {
			return nil, fmt.Errorf("iterating datacap balance hamt: %w", err)
		}
		return balances, nil
	}

	changes, err := datacap.DiffDataCapBalances(ctx, node.Store(), ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing datacap balances: %w", err)
	}

	for _, change := range changes.Added {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.Address.String(),
			Event:       datacapmodel.Added,
			DataCap:     change.DataCap.String(),
			AddressType: datacapmodel.VerifierClient,
		})

	}

	for _, change := range changes.Removed {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.Address.String(),
			Event:       datacapmodel.Removed,
			DataCap:     change.DataCap.String(),
			AddressType: datacapmodel.VerifierClient,
		})
	}

	for _, change := range changes.Modified {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.After.Address.String(),
			Event:       datacapmodel.Modified,
			DataCap:     change.After.DataCap.String(),
			AddressType: datacapmodel.VerifierClient,
		})
	}

	// Handle the verifreg
	verifregActor, actorErr := node.Actor(ctx, builtin.VerifiedRegistryActorAddr, a.Current.Key())
	if actorErr != nil {
		log.Errorf("get error during getting VerifiedRegistry: %v", actorErr)
	}
	currentVerifregState, _ := verifreg.Load(node.Store(), verifregActor)

	preVerifregActor, actorErr := node.Actor(ctx, builtin.VerifiedRegistryActorAddr, a.Executed.Key())
	if actorErr != nil {
		log.Errorf("get error during getting VerifiedRegistry: %v", actorErr)
	}
	preVerifregState, _ := verifreg.Load(node.Store(), preVerifregActor)

	verifierChanges, err := verifreg.DiffVerifiers(ctx, node.Store(), preVerifregState, currentVerifregState)
	if err != nil {
		return nil, fmt.Errorf("diffing verified registry verifiers: %w", err)
	}

	// a new verifier was added
	for _, change := range verifierChanges.Added {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.Address.String(),
			Event:       datacapmodel.Added,
			DataCap:     change.DataCap.String(),
			AddressType: datacapmodel.Verifier,
		})
	}
	// a verifier was removed
	for _, change := range verifierChanges.Removed {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.Address.String(),
			Event:       datacapmodel.Removed,
			DataCap:     change.DataCap.String(),
			AddressType: datacapmodel.Verifier,
		})
	}
	// an existing verifier's DataCap changed
	for _, change := range verifierChanges.Modified {
		balances = append(balances, &datacapmodel.DataCapBalance{
			Height:      int64(ec.CurrTs.Height()),
			StateRoot:   ec.CurrTs.ParentState().String(),
			Address:     change.After.Address.String(),
			Event:       datacapmodel.Modified,
			DataCap:     change.After.DataCap.String(),
			AddressType: datacapmodel.Verifier,
		})
	}
	return balances, nil
}

type DataCapExtractionContext struct {
	PrevState, CurrState datacap.State
	PrevTs, CurrTs       *types.TipSet

	PreviousStatePresent bool
}

func (d *DataCapExtractionContext) HasPreviousState() bool {
	return d.PreviousStatePresent
}

func NewBalanceExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*DataCapExtractionContext, error) {
	curState, err := datacap.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, fmt.Errorf("loading current datacap state: %w", err)
	}

	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		// actor doesn't exist yet, may have just been created.
		if err == types.ErrActorNotFound {
			return &DataCapExtractionContext{
				CurrState:            curState,
				PrevTs:               a.Executed,
				CurrTs:               a.Current,
				PrevState:            nil,
				PreviousStatePresent: false,
			}, nil
		}
		return nil, fmt.Errorf("loading previous datacap actor from parent tipset %s current height epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)

	}

	prevState, err := datacap.Load(node.Store(), prevActor)
	if err != nil {
		return nil, fmt.Errorf("loading previous datacap state: %w", err)
	}
	return &DataCapExtractionContext{
		PrevState:            prevState,
		CurrState:            curState,
		PrevTs:               a.Executed,
		CurrTs:               a.Current,
		PreviousStatePresent: true,
	}, nil
}
