package visor

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

const (
	ProcessingStatusOK    = "OK"
	ProcessingStatusInfo  = "INFO"  // Processing was successful but the task reported information in the StatusInformation column
	ProcessingStatusError = "ERROR" // one or more errors were encountered, data may be incomplete
)

type ProcessingReport struct {
	tableName struct{} `pg:"visor_processing_reports"` // nolint: structcheck,unused

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

func (p *ProcessingReport) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, p)
}

type ProcessingReportList []*ProcessingReport

func (pl ProcessingReportList) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(pl) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingReportList.Persist", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()

	return s.PersistModel(ctx, pl)
}
