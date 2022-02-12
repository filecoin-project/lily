package chain

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

func NewModelExporter(concurrency int) *ModelExporter {
	return &ModelExporter{persistSlot: make(chan struct{}, concurrency)}
}

type ModelExporter struct {
	persistSlot chan struct{} // filled with a token when a goroutine is persisting data
}

type ModelResults struct {
	Name  string
	Model model.PersistableList
}

func (me *ModelExporter) Export(ctx context.Context, strg model.Storage, models chan *ModelResults) error {
	for res := range models {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case me.persistSlot <- struct{}{}:
		}
		// wait until there is an empty slot before persisting
		start := time.Now()
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, res.Name))

		if err := strg.PersistBatch(ctx, res.Model); err != nil {
			stats.Record(ctx, metrics.PersistFailure.M(1))
			log.Errorw("persistence failed", "task", res.Name, "error", err)
			return err
		}
		log.Debugw("model data persisted", "task", res.Model, "duration", time.Since(start))
		<-me.persistSlot
	}
	return nil
}

func (me *ModelExporter) Close() error {
	log.Debug("closing model exporter")

	// We need to ensure that any persistence goroutine has completed. Since the channel has capacity 1 we can detect
	// when the persistence goroutine is running by attempting to send a probe value on the channel. When the channel
	// contains a token then we are still persisting and we should wait for that to be done.
	select {
	case me.persistSlot <- struct{}{}:
		// no token was in channel so there was no persistence goroutine running
	default:
		// channel contained a token so persistence goroutine is running
		// wait for the persistence to finish, which is when the channel can be sent on
		log.Debug("waiting for persistence to complete")
		me.persistSlot <- struct{}{}
		log.Debug("persistence completed")
	}

	// When we reach here there will always be a single token in the channel (our probe) which needs to be drained so
	// the channel is empty for reuse.
	<-me.persistSlot

	return nil
}
