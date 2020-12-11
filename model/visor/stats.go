package visor

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type ProcessingStat struct {
	tableName struct{} `pg:"visor_processing_stats"` // nolint: structcheck,unused

	// RecordedAt is the time the measurement was recorded in the database
	RecordedAt time.Time `pg:",pk,notnull"`

	// Measure is the name of the measurement, e.g. `completed_count`
	Measure string `pg:",pk,notnull"`

	// Tag is the subtype of the measurement, e.g. `tipsets_messages`
	Tag string `pg:",pk,notnull"`

	// Value is the value of the measurement
	Value int64 `pg:",use_zero,notnull"`
}

func (p *ProcessingStat) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, p)
}

type ProcessingStatList []*ProcessingStat

func (pl ProcessingStatList) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(pl) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingStatList.Persist", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()

	return s.PersistModel(ctx, pl)
}
