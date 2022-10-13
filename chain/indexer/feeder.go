package indexer

import (
	"context"
	"os"
	"time"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/market"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/miner"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/raw"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/block"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/economics"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/message"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
)

const BitWidth = 8

type Feeder struct {
	Strg model.Storage
}

func (f *Feeder) Index(ctx context.Context, path string) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()
	start := time.Now()
	mr, err := cborable.NewModelReader(ctx, fi)
	if err != nil {
		return err
	}

	for _, state := range mr.States() {
		tipset := state.Current
		parent := state.Executed

		tasks, err := mr.ModelMetasForTipSet(tipset.Key())
		if err != nil {
			return err
		}

		transformer, consumer, err := f.startRouters(ctx, tasks,
			[]transform.Handler{
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
				economics.NewGasEconomyTransform(),
				message.NewReceiptTransform(),

				block.NewBlockHeaderTransform(),
				block.NewBlockParentsTransform(),
				block.NewDrandBlockEntryTransform(),
			}, []load.Handler{
				&persistable.PersistableResultConsumer{Strg: f.Strg},
			})

		// TODO handle the error case here, remove the panic in the goroutine
		go func() {
			for _, task := range tasks {
				data, err := mr.GetModels(tipset.Key(), task)
				if err != nil {
					log.Errorw("getting models", "error", err)
					panic(err)
				}
				if err := transformer.Route(ctx, &resultImpl{
					task:     task,
					current:  tipset,
					executed: parent,
					complete: true,
					result: &extract.StateResult{
						Task:      task,
						Error:     nil,
						Data:      data,
						StartedAt: time.Now(),
						Duration:  0,
					},
				}); err != nil {
					log.Errorw("routing models", "error", err)
					panic(err)
				}
			}
			if err := transformer.Stop(); err != nil {
				log.Errorw("stopping transformer", "error", err)
			}
		}()

		for res := range transformer.Results() {
			if err := consumer.Route(ctx, res); err != nil {
				return err
			}
		}
		if err := consumer.Stop(); err != nil {
			return err
		}

	}
	log.Infow("index complete", "duration", time.Since(start))
	return nil
}

type resultImpl struct {
	task     v2.ModelMeta
	current  *types.TipSet
	executed *types.TipSet
	complete bool
	result   *extract.StateResult
}

func (r *resultImpl) Models() []v2.LilyModel {
	//TODO implement me
	panic("implement me")
}

func (r *resultImpl) Task() v2.ModelMeta {
	return r.task
}

func (r *resultImpl) Current() *types.TipSet {
	return r.current
}

func (r *resultImpl) Executed() *types.TipSet {
	return r.executed
}

func (r *resultImpl) Complete() bool {
	return r.complete
}

func (r *resultImpl) State() *extract.StateResult {
	return r.result
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

func (f *Feeder) startRouters(ctx context.Context, tasks []v2.ModelMeta, handlers []transform.Handler, consumers []load.Handler) (Transformer, Loader, error) {
	tr, err := transform.NewRouter(tasks, handlers...)
	if err != nil {
		return nil, nil, err
	}
	tr.Start(ctx)

	lr, err := load.NewRouter(consumers...)
	if err != nil {
		return nil, nil, err
	}
	lr.Start(ctx)

	return tr, lr, nil
}
