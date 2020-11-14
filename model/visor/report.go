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

const (
	ProcessingStatusOK    = "OK"
	ProcessingStatusInfo  = "INFO"  // Processing was successful but the task reported information in the StatusInformation column
	ProcessingStatusError = "ERROR" // one or more errors were encountered, data may be incomplete
)

type ProcessingReport struct {
	tableName struct{} `pg:"visor_processing_reports"`

	Height    int64  `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	// Reporter is the name of the instance that is reporting the result
	Reporter string `pg:",pk,notnull"`

	// Task is the name of the sub task that generated the report
	Task string `pg:",pk,notnull"`

	StartedAt   time.Time `pg:",pk,use_zero"`
	CompletedAt time.Time `pg:",use_zero"`

	Status            string `pg:",notnull"`
	StatusInformation string
	ErrorsDetected    interface{} `pg:",type:jsonb"`
}

func (s *ProcessingReport) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, s).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing report: %w", err)
	}
	return nil
}

type ProcessingReportList []*ProcessingReport

func (l ProcessingReportList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingReportList.PersistWithTx", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()

	if _, err := tx.ModelContext(ctx, &l).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing report: %w", err)
	}
	return nil
}
