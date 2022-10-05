package indexer

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	carbs "github.com/ipld/go-car/v2/blockstore"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/load"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/cborable"
	"github.com/filecoin-project/lily/chain/indexer/v2/load/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/raw"
	"github.com/filecoin-project/lily/model"
	v2 "github.com/filecoin-project/lily/model/v2"
	raw2 "github.com/filecoin-project/lily/model/v2/actors/raw"
	"github.com/filecoin-project/lily/tasks"
)

const BitWidth = 8

type Feeder struct {
	Api  tasks.DataSource
	Strg model.Storage
}

/*
	root, err := cid.Decode("bafy2bzacedinjodppy3tu5jjursdn4mrlr5e4cvzsdupc5axmkakiuojybkrq")
	if err != nil {
		return false, err
	}
	f := indexer.Feeder{Api: m.api, Strg: m.strg}
	if err := f.Index(ctx, ts, root); err != nil {
		panic(err)
	}
	return false, nil

*/

func (f *Feeder) Index(ctx context.Context, ts *types.TipSet, root cid.Cid) error {
	start := time.Now()
	ro, err := carbs.OpenReadOnly("./" + ts.ParentState().String() + ".car")
	if err != nil {
		return err
	}
	store := adt.WrapBlockStore(ctx, ro)

	stateMap, err := adt.AsMap(store, root, BitWidth)
	if err != nil {
		return err
	}
	var modelRoot typegen.CborCid
	if found, err := stateMap.Get(cborable.TipsetKeyer{T: ts}, &modelRoot); err != nil {
		return err
	} else if !found {
		panic("here")
	}
	modelMap, err := adt.AsMultimap(store, cid.Cid(modelRoot), BitWidth, BitWidth)
	if err != nil {
		return err
	}
	taskData := make(map[v2.ModelMeta]*adt.Array)
	var tasks []v2.ModelMeta
	if err := modelMap.ForAll(func(k string, arr *adt.Array) error {
		meta, err := v2.DecodeModelMeta(k)
		if err != nil {
			return err
		}
		taskData[meta] = arr
		tasks = append(tasks, meta)
		return nil
	}); err != nil {
		return err
	}

	transformer, consumer, err := f.startRouters(ctx, tasks,
		[]transform.Handler{
			raw.NewActorTransform(),
			raw.NewActorStateTransform(),
		}, []load.Handler{
			&persistable.PersistableResultConsumer{Strg: f.Strg},
		})

	go func() {
		for meta, arr := range taskData {
			// TODO we need a nice way to decode data loaded from the car before handing it to a transform
			actorState := &raw2.ActorState{}
			meta.Kind = v2.ModelActorKind
			switch meta {
			case actorState.Meta():
				var thisState raw2.ActorState
				var toRount = make([]v2.LilyModel, 0, 10)
				if err := arr.ForEach(&thisState, func(i int64) error {
					cp := thisState
					toRount = append(toRount, &cp)
					return nil
				}); err != nil {
					panic(err)
				}
				if err := transformer.Route(ctx, &resultImpl{
					task:     meta,
					current:  ts,
					executed: nil,
					complete: true,
					result: &extract.StateResult{
						Task:      meta,
						Error:     nil,
						Data:      toRount,
						StartedAt: time.Now(),
						Duration:  0,
					},
				}); err != nil {
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
			return err
		}
	}
	if err := consumer.Stop(); err != nil {
		return err
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
	tr.Start(ctx, f.Api)

	lr, err := load.NewRouter(consumers...)
	if err != nil {
		return nil, nil, err
	}
	lr.Start(ctx)

	return tr, lr, nil
}
