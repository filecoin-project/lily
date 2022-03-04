package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/queue"
)

type IndexHandler struct {
	im *indexer.Manager
}

func NewIndexHandler(m *indexer.Manager) *IndexHandler {
	return &IndexHandler{im: m}
}

func (ih *IndexHandler) HandleIndexTipSetTask(ctx context.Context, t *asynq.Task) error {
	var p IndexTipSetPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Infow("indexing tipset", "tipset", p.TipSet.String(), "height", p.TipSet.Height(), "tasks", p.Tasks)

	_, err := ih.im.TipSet(ctx, p.TipSet, p.Tasks...)
	if err != nil {
		return err
	}
	return nil
}

type TipSetWorker struct {
	name        string
	concurrency int
	cfg         *queue.RedisConfig
	mux         *asynq.ServeMux
	done        chan struct{}
}

func NewTipSetWorker(im *indexer.Manager, name string, concurrency int, cfg *queue.RedisConfig) *TipSetWorker {
	ih := NewIndexHandler(im)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeIndexTipSet, ih.HandleIndexTipSetTask)
	return &TipSetWorker{
		name:        name,
		concurrency: concurrency,
		cfg:         cfg,
		mux:         mux,
	}
}

func (t *TipSetWorker) Run(ctx context.Context) error {
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
			Logger:      log.With("process", fmt.Sprintf("TipSetWorker-%s", t.name)),
			LogLevel:    asynq.InfoLevel,
			Queues: map[string]int{
				string(High):   3,
				string(Medium): 2,
				string(Low):    1,
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

func (t *TipSetWorker) Done() <-chan struct{} {
	return t.done
}
