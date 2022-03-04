package asynq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/index/integrated"
	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/asynq")

type AsynqTipSetTaskHandler struct {
	im *integrated.Manager
}

func NewIndexHandler(m *integrated.Manager) *AsynqTipSetTaskHandler {
	return &AsynqTipSetTaskHandler{im: m}
}

func (ih *AsynqTipSetTaskHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("indexing tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	_, err := ih.im.TipSet(ctx, p.TipSet, "")
	if err != nil {
		return err
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

func NewAsynqWorker(im *integrated.Manager, name string, concurrency int, cfg config.AsynqRedisConfig) *AsynqWorker {
	ih := NewIndexHandler(im)
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
