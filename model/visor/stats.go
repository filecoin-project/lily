package visor

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type ProcessingStat struct {
	tableName struct{} `pg:"visor_processing_stats"`

	// RecordedAt is the time the measurement was recorded in the database
	RecordedAt time.Time `pg:",pk,notnull"`

	// Measure is the name of the measurement, e.g. `messages_completed_count`
	Measure string `pg:",pk,notnull"`

	// Value is the value of the measurement
	Value int64 `pg:",use_zero,notnull"`
}

func (s *ProcessingStat) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, s).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing stat: %w", err)
	}
	return nil
}

type ProcessingStatList []*ProcessingStat

func (l ProcessingStatList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingStatList.PersistWithTx", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()

	if _, err := tx.ModelContext(ctx, &l).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing stats: %w", err)
	}
	return nil
}
