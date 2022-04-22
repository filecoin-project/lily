package queue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

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
		// TODO configure error handling
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
			StrictPriority: true,
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
