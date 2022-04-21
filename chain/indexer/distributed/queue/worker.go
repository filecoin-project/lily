package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/asynq")

type AsynqTipSetTaskHandler struct {
	indexer indexer.Indexer
}

func NewIndexHandler(i indexer.Indexer) *AsynqTipSetTaskHandler {
	return &AsynqTipSetTaskHandler{indexer: i}
}

func (ih *AsynqTipSetTaskHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("indexing tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	success, err := ih.indexer.TipSet(ctx, p.TipSet, "", p.Tasks...)
	if err != nil {
		return err
	}
	if !success {
		log.Warnw("failed to index task successfully", "height", p.TipSet.Height(), "tipset", p.TipSet.Key().String())
	}
	return nil
}

type AsynqWorker struct {
	name        string
	concurrency int
	cfg         config.AsynqRedisConfig
	mux         *asynq.ServeMux
	done        chan struct{}
}

func NewAsynqWorker(i indexer.Indexer, name string, concurrency int, cfg config.AsynqRedisConfig) *AsynqWorker {
	ih := NewIndexHandler(i)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeIndexTipSet, ih.HandleIndexTipSetTask)
	return &AsynqWorker{
		name:        name,
		concurrency: concurrency,
		cfg:         cfg,
		mux:         mux,
	}
}

const (
	WatcherQueue = "WATCHER"
	WalkerQueue  = "WALKER"
	IndexQueue   = "INDEX"
)

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
				WatcherQueue: 3,
				WalkerQueue:  2,
				IndexQueue:   1,
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
