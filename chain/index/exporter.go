package index

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

var mdlLog = logging.Logger("lily/index/exporter")

func NewModelExporter() *ModelExporter {
	return &ModelExporter{}
}

type ModelExporter struct{}

type ModelResult struct {
	Name  string
	Model model.Persistable
}

// ExportResult synchronously persists ModelResult `res` to model.Storage `strg`. An error is returned if persisting the
// model fails.
func (me *ModelExporter) ExportResult(ctx context.Context, strg model.Storage, res *ModelResult) error {
	ctx, span := otel.Tracer("").Start(ctx, "ModelExporter.ExportResult")
	defer span.End()
	start := time.Now()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, res.Name))

	if err := strg.PersistBatch(ctx, res.Model); err != nil {
		stats.Record(ctx, metrics.PersistFailure.M(1))
		return err
	}
	mdlLog.Infow("model data persisted", "task", res.Name, "duration", time.Since(start))
	return nil
}
