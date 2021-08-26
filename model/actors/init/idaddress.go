package init

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

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

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
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

	metrics.RecordCount(ctx, metrics.PersistModel, len(ias))
	return s.PersistModel(ctx, ias)
}
