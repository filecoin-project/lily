package indexer

import (
	"context"
	"fmt"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/keymutex"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

var mdlLog = logging.Logger("lily/index/exporter")

func NewModelExporter() *ModelExporter {
	return &ModelExporter{
		heightKeyMu: keymutex.NewHashed(907), // a prime greater than finality.
	}
}

type ModelExporter struct {
	heightKeyMu keymutex.KeyMutex
}

type ModelResult struct {
	Name  string
	Model model.Persistable
}

// ExportResult persists []ModelResult `results` to model.Storage `strg`. An error is returned if persisting the
// model fails. This method will block if models at `height` are being persisted allowing the following constraints to be met:
// - if data with height N and SR1 is being persisted and a request to persist data with the same values is made, allow it
// - if data with height N and SR2 is being persisted and a request to persist data with height N and SR1 is made, block
func (me *ModelExporter) ExportResult(ctx context.Context, strg model.Storage, height int64, results []*ModelResult) error {
	// lock exporting based on height only allowing a single height to be persisted simultaneously
	heightKey := strconv.FormatInt(height, 10)
	me.heightKeyMu.LockKey(heightKey)
	if err := me.heightKeyMu.UnlockKey(heightKey); err != nil {
		//NB: this could be a panic or ignored since it would indicate some fundamentally impossible error, the lock will always exist given the prior lock call.
		return err
	}

	grp, ctx := errgroup.WithContext(ctx)
	for _, res := range results {
		res := res

		grp.Go(func() error {
			ctx, span := otel.Tracer("").Start(ctx, fmt.Sprintf("ModelExporter.ExportResult.%s", res.Name))
			defer span.End()
			start := time.Now()
			ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, res.Name))

			if err := strg.PersistBatch(ctx, res.Model); err != nil {
				stats.Record(ctx, metrics.PersistFailure.M(1))
				return err
			}
			mdlLog.Infow("model data persisted", "height", height, "task", res.Name, "duration", time.Since(start))
			return nil
		})
	}
	return grp.Wait()
}
