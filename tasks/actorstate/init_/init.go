package init_

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var log = logging.Logger("lily/tasks/init")

// InitExtractor extracts init actor state
type InitExtractor struct{}

func (InitExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "InitExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "InitExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	stop := metrics.Timer(ctx, metrics.StateExtractionDuration)
	defer stop()

	// genesis state.
	if a.Current.Height() == 1 {
		initActorState, err := init_.Load(node.Store(), &a.Actor)
		if err != nil {
			return nil, err
		}

		out := initmodel.IdAddressList{}
		for _, builtinAddress := range []address.Address{
			builtin.SystemActorAddr, builtin.InitActorAddr,
			builtin.RewardActorAddr, builtin.CronActorAddr, builtin.StoragePowerActorAddr, builtin.StorageMarketActorAddr,
			builtin.VerifiedRegistryActorAddr, builtin.BurntFundsActorAddr,
		} {
			out = append(out, &initmodel.IdAddress{
				Height:    0,
				ID:        builtinAddress.String(),
				Address:   builtinAddress.String(),
				StateRoot: a.Executed.ParentState().String(),
			})
		}
		if err := initActorState.ForEachActor(func(id abi.ActorID, addr address.Address) error {
			idAddr, err := address.NewIDAddress(uint64(id))
			if err != nil {
				return err
			}
			out = append(out, &initmodel.IdAddress{
				Height:    int64(a.Current.Height()),
				ID:        idAddr.String(),
				Address:   addr.String(),
				StateRoot: a.Current.ParentState().String(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}
	prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
	if err != nil {
		return nil, xerrors.Errorf("loading previous init actor: %w", err)
	}

	prevState, err := init_.Load(node.Store(), prevActor)
	if err != nil {
		return nil, xerrors.Errorf("loading previous init actor state: %w", err)
	}

	curState, err := init_.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current init actor state: %w", err)
	}

	addressChanges, err := init_.DiffAddressMap(ctx, node.Store(), prevState, curState)
	if err != nil {
		return nil, xerrors.Errorf("diffing init actor state: %w", err)
	}

	out := make(initmodel.IdAddressList, 0, len(addressChanges.Added)+len(addressChanges.Modified))
	for _, newAddr := range addressChanges.Added {
		out = append(out, &initmodel.IdAddress{
			Height:    int64(a.Current.Height()),
			StateRoot: a.Current.ParentState().String(),
			ID:        newAddr.ID.String(),
			Address:   newAddr.PK.String(),
		})
	}
	for _, modAddr := range addressChanges.Modified {
		out = append(out, &initmodel.IdAddress{
			Height:    int64(a.Current.Height()),
			StateRoot: a.Current.ParentState().String(),
			ID:        modAddr.To.ID.String(),
			Address:   modAddr.To.PK.String(),
		})
	}

	return out, nil
}
