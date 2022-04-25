package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/trace"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/storage"
)

var log = logging.Logger("lily/asynq")

type AsynqWorker struct {
	name        string
	concurrency int
	cfg         config.AsynqRedisConfig
	mux         *asynq.ServeMux
	done        chan struct{}
}

func NewAsynqWorker(i indexer.Indexer, db *storage.Database, name string, concurrency int, cfg config.AsynqRedisConfig) *AsynqWorker {
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeIndexTipSet, tasks.NewIndexHandler(i).HandleIndexTipSetTask)
	mux.HandleFunc(tasks.TypeGapFillTipSet, tasks.NewGapFillHandler(i, db).HandleGapFillTipSetTask)
	return &AsynqWorker{
		name:        name,
		concurrency: concurrency,
		cfg:         cfg,
		mux:         mux,
	}
}

func (t *AsynqWorker) Run(ctx context.Context) error {
	t.done = make(chan struct{})
	defer close(t.done)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Network:  t.cfg.Network,
			Addr:     t.cfg.Addr,
			Username: t.cfg.Username,
			Password: t.cfg.Password,
			DB:       t.cfg.DB,
			PoolSize: t.cfg.PoolSize,
		},
		asynq.Config{
			Concurrency: t.concurrency,
			Logger:      log.With("process", fmt.Sprintf("AsynqWorker-%s", t.name)),
			LogLevel:    asynq.DebugLevel,
			Queues: map[string]int{
				indexer.Watch.String(): 6,
				indexer.Walk.String():  2,
				indexer.Index.String(): 1,
				indexer.Fill.String():  1,
			},
			StrictPriority: false,
			ErrorHandler:   &WorkerErrorHandler{},
		},
	)
	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()
	return srv.Run(t.mux)
}

func (t *AsynqWorker) Done() <-chan struct{} {
	return t.done
}

type WorkerErrorHandler struct {
}

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
