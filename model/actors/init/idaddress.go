package init

import (
	"context"
	"fmt"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

// IDAddress contains a mapping of ID addresses to robust addresses from the init actor’s state.
type IDAddress struct {
	tableName struct{} `pg:"id_addresses"` // nolint: structcheck
	// Epoch when this address mapping was created or updated.
	Height int64 `pg:",pk,notnull,use_zero"`
	// ID address
	ID string `pg:",pk,notnull"`
	// Robust address
	Address string `pg:",pk,notnull"`
	// StateRoot when this address mapping was created or updated.
	StateRoot string `pg:",pk,notnull"`
}

type IDAddressV0 struct {
	tableName struct{} `pg:"id_addresses"` // nolint: structcheck
	ID        string   `pg:",pk,notnull"`
	Address   string   `pg:",pk,notnull"`
	StateRoot string   `pg:",pk,notnull"`
}

func (ia *IDAddress) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if ia == nil {
			return (*IDAddressV0)(nil), true
		}

		return &IDAddressV0{
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

func (ia *IDAddress) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "id_addresses"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := ia.AsVersion(version)
	if !ok {
		return fmt.Errorf("IDAddress not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type IDAddressList []*IDAddress

func (ias IDAddressList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "IDAddressList.PersistWithTx")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ias)))
	}
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

	metrics.RecordCount(ctx, metrics.PersistModel, len(ias))
	return s.PersistModel(ctx, ias)
}
