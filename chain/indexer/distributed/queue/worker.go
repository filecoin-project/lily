package queue

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/distributed"
	"github.com/filecoin-project/lily/chain/indexer/distributed/queue/tasks"
	"github.com/filecoin-project/lily/storage"
)

type AsynqWorker struct {
	done   chan struct{}
	server *distributed.TipSetWorker
	index  indexer.Indexer
	db     *storage.Database
}

func NewAsynqWorker(i indexer.Indexer, db *storage.Database, server *distributed.TipSetWorker) *AsynqWorker {
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

	if err := t.server.Run(mux); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		t.server.Shutdown()
	}()

	return nil
}

func (t *AsynqWorker) Done() <-chan struct{} {
	return t.done
}
