package v2

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/tasks"
)

type Manager struct {
	indexer *TipSetIndexer
	tasks   []v2.ModelMeta
	api     tasks.DataSource
}

func NewIndexManager(api tasks.DataSource, tasks []v2.ModelMeta) *Manager {
	return &Manager{
		indexer: NewTipSetIndexer(api, tasks, 64),
		tasks:   tasks,
		api:     api,
	}
}

func (m *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (bool, error) {
	results, err := m.indexer.TipSet(ctx, ts)
	if err != nil {
		return false, err
	}

	emitter, consumer, err := m.startRouters(ctx,
		[]Handler{NewSectorInfoToPostgresHandler()},
		[]ResultConsumer{&PersistableResultConsumer{strg: storage.NewMemStorageLatest()}})
	if err != nil {
		return false, err
	}

	go func() {
		for res := range results {
			if err := emitter.Emit(ctx, res); err != nil {
				panic(err)
			}
		}
		emitter.Stop()
	}()
	for res := range emitter.Results() {
		if err := consumer.Emit(ctx, res); err != nil {
			return false, err
		}
	}
	consumer.Stop()
	return true, nil
}

type Emitter interface {
	// put things in here to process them
	Emit(ctx context.Context, data *TipSetResult) error
	// processed items come out here
	Results() chan HandlerResult
	Stop()
}

type Consumer interface {
	Emit(ctx context.Context, data HandlerResult) error
	Stop()
}

func (m *Manager) startRouters(ctx context.Context, handlers []Handler, consumers []ResultConsumer) (Emitter, Consumer, error) {
	hr, err := NewHandlerRouter()
	if err != nil {
		return nil, nil, err
	}
	for _, handler := range handlers {
		hr.AddHandler(handler)
	}
	hr.Start(ctx, m.api)

	rr, err := NewResultRouter()
	if err != nil {
		return nil, nil, err
	}
	for _, consumer := range consumers {
		rr.AddConsumer(consumer)
	}
	rr.Start(ctx)

	return hr, rr, nil
}
