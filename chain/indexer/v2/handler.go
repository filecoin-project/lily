package v2

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/miner"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/message"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

type Manager struct {
	indexer *TipSetIndexer
	tasks   []v2.ModelMeta
	api     tasks.DataSource
	strg    model.Storage
}

func NewIndexManager(strg model.Storage, api tasks.DataSource, tasks []v2.ModelMeta) *Manager {
	return &Manager{
		indexer: NewTipSetIndexer(api, tasks, 64),
		tasks:   tasks,
		api:     api,
		strg:    strg,
	}
}

func (m *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (bool, error) {
	results, err := m.indexer.TipSet(ctx, ts)
	if err != nil {
		return false, err
	}

	router, consumer, err := m.startRouters(ctx,
		[]transform.Handler{miner.NewSectorInfoTransform(), message.NewVMMessageTransform()},
		[]load.Handler{&persistable.PersistableResultConsumer{Strg: m.strg}})
	if err != nil {
		return false, err
	}

	go func() {
		for res := range results {
			if err := router.Route(ctx, res); err != nil {
				panic(err)
			}
		}
		router.Stop()
	}()
	for res := range router.Results() {
		if err := consumer.Route(ctx, res); err != nil {
			return false, err
		}
	}
	consumer.Stop()
	return true, nil
}

type Transformer interface {
	Route(ctx context.Context, data transform.IndexState) error
	Results() chan transform.Result
	Stop()
}

type Loader interface {
	Route(ctx context.Context, data transform.Result) error
	Stop()
}

func (m *Manager) startRouters(ctx context.Context, handlers []transform.Handler, consumers []load.Handler) (Transformer, Loader, error) {
	tr, err := transform.NewRouter(handlers...)
	if err != nil {
		return nil, nil, err
	}
	tr.Start(ctx, m.api)

	lr, err := load.NewRouter(consumers...)
	if err != nil {
		return nil, nil, err
	}
	lr.Start(ctx)

	return tr, lr, nil
}
