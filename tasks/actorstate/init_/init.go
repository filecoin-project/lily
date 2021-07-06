package init_

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model/registry"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	init_ "github.com/filecoin-project/sentinel-visor/chain/actors/builtin/init"
	"github.com/filecoin-project/sentinel-visor/model"
)

const ActorStatesInitTask = "actorstatesinit" // task that only extracts init actor states (but not the raw state)

func init() {
	for _, c := range init_.AllCodes() {
		actor.Register(c, InitExtractor{})
	}
	registry.ModelRegistry.Register(ActorStatesInitTask, &IdAddress{})
}

// InitExtractor extracts init actor state
type InitExtractor struct{}

func (InitExtractor) Extract(ctx context.Context, a actor.ActorInfo, node actor.ActorStateAPI) (model.Persistable, error) {
	// genesis state.
	if a.Epoch == 1 {
		initActorState, err := init_.Load(node.Store(), &a.Actor)
		if err != nil {
			return nil, err
		}

		out := IdAddressList{}
		for _, builtinAddress := range []address.Address{builtin.SystemActorAddr, builtin.InitActorAddr,
			builtin.RewardActorAddr, builtin.CronActorAddr, builtin.StoragePowerActorAddr, builtin.StorageMarketActorAddr,
			builtin.VerifiedRegistryActorAddr, builtin.BurntFundsActorAddr} {
			out = append(out, &IdAddress{
				Height:    0,
				ID:        builtinAddress.String(),
				Address:   builtinAddress.String(),
				StateRoot: a.ParentTipSet.ParentState().String(),
			})
		}
		if err := initActorState.ForEachActor(func(id abi.ActorID, addr address.Address) error {
			idAddr, err := address.NewIDAddress(uint64(id))
			if err != nil {
				return err
			}
			out = append(out, &IdAddress{
				Height:    int64(a.Epoch),
				ID:        idAddr.String(),
				Address:   addr.String(),
				StateRoot: a.ParentStateRoot.String(),
			})
			return nil
		}); err != nil {
			return nil, err
		}
		return out, nil
	}
	prevActor, err := node.StateGetActor(ctx, a.Address, a.ParentTipSet.Key())
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

	out := make(IdAddressList, 0, len(addressChanges.Added)+len(addressChanges.Modified))
	for _, newAddr := range addressChanges.Added {
		out = append(out, &IdAddress{
			Height:    int64(a.Epoch),
			StateRoot: a.ParentStateRoot.String(),
			ID:        newAddr.ID.String(),
			Address:   newAddr.PK.String(),
		})
	}
	for _, modAddr := range addressChanges.Modified {
		out = append(out, &IdAddress{
			Height:    int64(a.Epoch),
			StateRoot: a.ParentStateRoot.String(),
			ID:        modAddr.To.ID.String(),
			Address:   modAddr.To.PK.String(),
		})
	}

	return out, nil
}

type IdAddress struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	ID        string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`
}

type IdAddressV0 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"id_addresses"`
	ID        string   `pg:",pk,notnull"`
	Address   string   `pg:",pk,notnull"`
	StateRoot string   `pg:",pk,notnull"`
}

func (ia *IdAddress) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if ia == nil {
			return (*IdAddressV0)(nil), true
		}

		return &IdAddressV0{
			ID:        ia.ID,
			Address:   ia.Address,
			StateRoot: ia.StateRoot,
		}, true
	case 1:
		return ia, true
	default:
		return nil, false
	}
}

func (ia *IdAddress) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "id_addresses"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := ia.AsVersion(version)
	if !ok {
		return xerrors.Errorf("IdAddress not supported for schema version %s", version)
	}

	return s.PersistModel(ctx, m)
}

type IdAddressList []*IdAddress

func (ias IdAddressList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "IdAddressList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ias))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "id_addresses"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range ias {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	return s.PersistModel(ctx, ias)
}
