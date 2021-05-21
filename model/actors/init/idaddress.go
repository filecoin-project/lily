package init

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type IdAddress struct {
	ID        string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`
}

func (ia *IdAddress) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "id_addresses"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, ia)
}

type IdAddressList []*IdAddress

func (ias IdAddressList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "IdAddressList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ias))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "id_addresses"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	for _, ia := range ias {
		if err := s.PersistModel(ctx, ia); err != nil {
			return err
		}
	}
	return nil
}
