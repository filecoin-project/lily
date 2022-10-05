package cborable

import (
	"context"
	"os"
	"reflect"
	"sort"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	carbs "github.com/ipld/go-car/v2/blockstore"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/cborable"
	v2 "github.com/filecoin-project/lily/model/v2"
)

var log = logging.Logger("load/car")

type CarResultConsumer struct {
}

func (c *CarResultConsumer) Name() string {
	return reflect.TypeOf(CarResultConsumer{}).Name()
}

func (c *CarResultConsumer) Type() transform.Kind {
	return "cborable"
}

func (c *CarResultConsumer) Consume(ctx context.Context, in chan transform.Result) error {
	modelStore, err := newModelStore(ctx, blockstore.NewMemory())
	if err != nil {
		return err
	}
	var ts *types.TipSet
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if res.Data() == nil {
				continue
			}
			models := res.Data().(cborable.CborablResult)
			ts = models.TipSet
			for _, m := range models.Model {
				if err := modelStore.PutModel(ctx, m); err != nil {
					return err
				}
			}
		}
	}
	stateRoot, err := modelStore.FinalizeModelMap(ctx, ts)
	if err != nil {
		return err
	}
	log.Infow("finalized model map", "root", stateRoot.String())
	f, err := os.Create("./" + ts.ParentState().String() + ".car")
	if err != nil {
		return err
	}
	if err := modelStore.AsCAR(ctx, f); err != nil {
		return err
	}
	return f.Close()
}

type ModelKeyer struct {
	M v2.LilyModel
}

func (m ModelKeyer) Key() string {
	return m.M.Meta().String()
}

type TipsetKeyer struct {
	T *types.TipSet
}

func (m TipsetKeyer) Key() string {
	return m.T.Key().String()
}

const BitWidth = 8

func newModelStore(ctx context.Context, bs blockstore.Blockstore) (*modelStore, error) {
	store := adt.WrapBlockStore(ctx, bs)
	stateMap, err := adt.MakeEmptyMap(store, BitWidth)
	if err != nil {
		return nil, err
	}
	modelMap, err := adt.MakeEmptyMultimap(store, BitWidth, BitWidth)
	if err != nil {
		return nil, err
	}
	return &modelStore{
		bs:       bs,
		store:    store,
		modelMap: modelMap,
		stateMap: stateMap,
		cache:    make([]v2.LilyModel, 0, 100),
	}, nil
}

type modelStore struct {
	bs    blockstore.Blockstore
	store adt.Store
	// map[ModelType][]Model
	modelMap *adt.Multimap
	// map[tipset?]map[modelType][]Model
	stateMap *adt.Map
	cache    []v2.LilyModel
}

func (c *modelStore) AsCAR(ctx context.Context, f *os.File) error {
	carRoot, err := c.stateMap.Root()
	if err != nil {
		return err
	}
	carrw, err := carbs.OpenReadWriteFile(f, []cid.Cid{carRoot}, carbs.WriteAsCarV1(true))
	if err != nil {
		return err
	}
	keys, err := c.bs.AllKeysChan(ctx)
	if err != nil {
		return err
	}
	for key := range keys {
		blk, err := c.bs.Get(ctx, key)
		if err != nil {
			return err
		}
		if err := carrw.Put(ctx, blk); err != nil {
			return err
		}
	}
	if err := carrw.Finalize(); err != nil {
		return err
	}
	actualCarRoot, err := carrw.Roots()
	if err != nil {
		return err
	}
	if !actualCarRoot[0].Equals(carRoot) {
		panic("here")
	}

	return nil
}

func (c *modelStore) FinalizeModelMap(ctx context.Context, ts *types.TipSet) (cid.Cid, error) {
	log.Infow("finalizing model map", "tipset", ts.Key().String())
	// deterministic ordering
	sort.Slice(c.cache, func(i, j int) bool {
		return c.cache[i].Cid().String() < c.cache[j].Cid().String()
	})
	for _, d := range c.cache {
		if err := c.modelMap.Add(ModelKeyer{d}, d); err != nil {
			return cid.Undef, err
		}
	}
	c.modelMap.ForAll(func(k string, arr *adt.Array) error {
		r, err := arr.Root()
		if err != nil {
			return err
		}
		log.Infow("modelMap", "model", k, "root", r.String())
		return nil
	})
	modelRoot, err := c.modelMap.Root()
	if err != nil {
		return cid.Undef, err
	}
	if err = c.stateMap.Put(TipsetKeyer{ts}, typegen.CborCid(modelRoot)); err != nil {
		return cid.Undef, err
	}

	log.Infow("finalized model root", "root", modelRoot.String())
	return c.stateMap.Root()
}

func (c *modelStore) PutModel(ctx context.Context, m v2.LilyModel) error {
	c.cache = append(c.cache, m)
	return nil
}
