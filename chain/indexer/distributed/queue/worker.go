package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/storage"
)

var log = logging.Logger("lily/distributed/worker")

type AsynqWorker struct {
	done chan struct{}

	name   string
	server *distributed.TipSetWorker
	index  indexer.Indexer
	db     *storage.Database
}

func NewAsynqWorker(name string, i indexer.Indexer, db *storage.Database, server *distributed.TipSetWorker) *AsynqWorker {
	return &AsynqWorker{
		name:   name,
		server: server,
		index:  i,
		db:     db,
	}
}

func (t *AsynqWorker) Run(ctx context.Context) error {
	t.done = make(chan struct{})
	defer close(t.done)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeIndexTipSet, tasks.NewIndexHandler(t.index).HandleIndexTipSetTask)
	mux.HandleFunc(tasks.TypeGapFillTipSet, tasks.NewGapFillHandler(t.index, t.db).HandleGapFillTipSetTask)

	t.server.ServerConfig.Logger = log.With("name", t.name)
	t.server.ServerConfig.ErrorHandler = &WorkerErrorHandler{}

	server := asynq.NewServer(t.server.RedisConfig, t.server.ServerConfig)
	if err := server.Start(mux); err != nil {
		return err
	}
	<-ctx.Done()
	server.Shutdown()
	return nil
}

func (t *AsynqWorker) Done() <-chan struct{} {
	return t.done
}

type WorkerErrorHandler struct{}

func (w *WorkerErrorHandler) HandleError(ctx context.Context, task *asynq.Task, err error) {
	switch task.Type() {
	case tasks.TypeIndexTipSet:
		var p tasks.IndexTipSetPayload
		if err := json.Unmarshal(task.Payload(), &p); err != nil {
			log.Errorw("failed to decode task type (developer error?)", "error", err)
		}
		if p.HasTraceCarrier() {
			if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
				ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
				trace.SpanFromContext(ctx).RecordError(err)
			}
		}
		log.Errorw("task failed", "type", task.Type(), "tipset", p.TipSet.Key().String(), "height", p.TipSet.Height(), "tasks", p.Tasks, "error", err)
	case tasks.TypeGapFillTipSet:
		var p tasks.GapFillTipSetPayload
		if err := json.Unmarshal(task.Payload(), &p); err != nil {
			log.Errorw("failed to decode task type (developer error?)", "error", err)
		}
		if p.HasTraceCarrier() {
			if sc := p.TraceCarrier.AsSpanContext(); sc.IsValid() {
				ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
				trace.SpanFromContext(ctx).RecordError(err)
			}
		}
		log.Errorw("task failed", "type", task.Type(), "tipset", p.TipSet.Key().String(), "height", p.TipSet.Height(), "tasks", p.Tasks, "error", err)
	}
}
