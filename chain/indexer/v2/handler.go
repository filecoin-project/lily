package v2

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	cborable2 "github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/market"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/miner"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/raw"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/block"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/message"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("indexmanager")

type Manager struct {
	indexer *TipSetIndexer
	tasks   []v2.ModelMeta
	api     tasks.DataSource
	strg    model.Storage
}

func NewIndexManager(strg model.Storage, api tasks.DataSource, tasks []v2.ModelMeta) *Manager {
	return &Manager{
		indexer: NewTipSetIndexer(api, tasks, 1024),
		tasks:   tasks,
		api:     api,
		strg:    strg,
	}
}

func (m *Manager) TipSet(ctx context.Context, ts *types.TipSet, options ...indexer.Option) (bool, error) {
	start := time.Now()
	results, err := m.indexer.TipSet(ctx, ts)
	if err != nil {
		return false, err
	}

	transformer, consumer, err := m.startRouters(ctx,
		[]transform.Handler{
			cborable.NewCborTransform(),

			raw.NewActorTransform(),
			raw.NewActorStateTransform(),

			miner.NewSectorInfoTransform(),
			miner.NewPrecommitEventTransformer(),
			miner.NewSectorEventTransformer(),
			miner.NewSectorDealsTransformer(),
			miner.NewPrecommitInfoTransformer(),

			market.NewDealProposalTransformer(),

			message.NewVMMessageTransform(),
			message.NewMessageTransform(),
			message.NewParsedMessageTransform(),
			message.NewBlockMessageTransform(),
			message.NewGasOutputTransform(),
			message.NewGasEconomyTransform(),
			message.NewReceiptTransform(),

			block.NewBlockHeaderTransform(),
			block.NewBlockParentsTransform(),
			block.NewDrandBlockEntryTransform(),
		},
		[]load.Handler{
			&persistable.PersistableResultConsumer{Strg: m.strg},
			&cborable2.CarResultConsumer{}},
	)
	if err != nil {
		return false, err
	}

	// TODO handle the error case here, remove the panic in the goroutine
	// - a simple solution would be to collect all transformer results and then send them to the consumer.
	//	 this will prevent partial persistence at the cost of more memory.
	go func() {
		for res := range results {
			if len(res.State().Data) > 0 {
				if err := transformer.Route(ctx, res); err != nil {
					panic(err)
				}
			}
		}
		if err := transformer.Stop(); err != nil {
			panic(err)
		}
	}()
	for res := range transformer.Results() {
		if err := consumer.Route(ctx, res); err != nil {
			return false, err
		}
	}
	if err := consumer.Stop(); err != nil {
		return false, err
	}
	log.Infow("index complete", "duration", time.Since(start))
	return true, nil
}

type Transformer interface {
	Route(ctx context.Context, data transform.IndexState) error
	Results() chan transform.Result
	Stop() error
}

type Loader interface {
	Route(ctx context.Context, data transform.Result) error
	Stop() error
}

func (m *Manager) startRouters(ctx context.Context, handlers []transform.Handler, consumers []load.Handler) (Transformer, Loader, error) {
	tr, err := transform.NewRouter(m.tasks, handlers...)
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
