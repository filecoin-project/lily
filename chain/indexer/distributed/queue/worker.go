package queue

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/storage"
)

type AsynqWorker struct {
	done   chan struct{}
	server *asynq.Server
	index  indexer.Indexer
	db     *storage.Database
}

func NewAsynqWorker(i indexer.Indexer, db *storage.Database, server *asynq.Server) *AsynqWorker {
	return &AsynqWorker{
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

	go func() {
		<-ctx.Done()
		t.server.Shutdown()
	}()
	return t.server.Run(mux)
}

func (t *AsynqWorker) Done() <-chan struct{} {
	return t.done
}
