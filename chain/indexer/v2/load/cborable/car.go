package cborable

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car/v2"
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
	if len(modelStore.cache) > 0 {
		stateRoot, err := modelStore.FinalizeModelMap(ctx, ts)
		if err != nil {
			return err
		}
		log.Infow("finalized model map", "root", stateRoot.String())
		f, err := os.Create(fmt.Sprintf("./%d_%s.car", ts.Height(), ts.ParentState()))
		if err != nil {
			return err
		}
		if err := modelStore.AsCAR(ctx, f); err != nil {
			return err
		}
		return f.Close()
	}
	return nil
}

type ModelKeyer struct {
	M v2.ModelMeta
}

func (m ModelKeyer) Key() string {
	return m.M.String()
}

type TipsetKeyer struct {
	T types.TipSetKey
}

func (m TipsetKeyer) Key() string {
	return m.T.String()
}

type CarModelStore struct {
	tsModelMmap map[types.TipSetKey]*adt.Multimap
	ro          *carbs.ReadOnly
}

func NewModelStoreFromCAR(ctx context.Context, path string) (*CarModelStore, error) {
	// open car file as read only blockstore
	r, err := car.OpenReader(path)
	if err != nil {
		return nil, err
	}
	stat, err := r.Inspect(true)
	if err != nil {
		return nil, err
	}
	log.Infow("stats", "info", stat)
	if err := r.Close(); err != nil {
		return nil, err
	}
	ro, err := carbs.OpenReadOnly(path)
	if err != nil {
		return nil, err
	}
	// file must have single root CID: the cid of the hamt.
	root, err := ro.Roots()
	if err != nil {
		return nil, err
	}
	if len(root) != 1 {
		return nil, fmt.Errorf("unrecognized car header root, expect 1, got %d", len(root))
	}
	// wrap the blockstore as an IPLD store
	store := adt.WrapBlockStore(ctx, ro)
	// load the root to get state map
	stateMap, err := adt.AsMap(store, root[0], BitWidth)
	if err != nil {
		return nil, err
	}
	var modelRoot typegen.CborCid
	tsModelMultiMap := make(map[types.TipSetKey]*adt.Multimap)
	if err := stateMap.ForEach(&modelRoot, func(key string) error {
		// TODO fix this, we should store the key as bytes
		key = strings.Replace(key, "{", "", -1)
		key = strings.Replace(key, "}", "", -1)
		cids, err := ParseTipSetString(key)
		if err != nil {
			return err
		}
		if len(cids) == 0 {
			log.Error("empty tipset")
			panic("here")
			return nil
		}
		modelMmap, err := adt.AsMultimap(store, cid.Cid(modelRoot), BitWidth, BitWidth)
		if err != nil {
			return err
		}
		k := types.NewTipSetKey(cids...)
		tsModelMultiMap[k] = modelMmap
		return nil
	}); err != nil {
		return nil, err
	}

	return &CarModelStore{tsModelMultiMap, ro}, nil
}

func (ms *CarModelStore) ModelMultiMapForTipSet(key types.TipSetKey) (*adt.Multimap, error) {
	models, found := ms.tsModelMmap[key]
	if !found {
		return nil, fmt.Errorf("no models for tipset %s", key)
	}
	return models, nil
}

func (ms *CarModelStore) ModelTasksForTipSet(key types.TipSetKey) ([]v2.ModelMeta, error) {
	models, err := ms.ModelMultiMapForTipSet(key)
	if err != nil {
		return nil, err
	}

	var tasks []v2.ModelMeta
	if err := models.ForAll(func(k string, arr *adt.Array) error {
		meta, err := v2.DecodeModelMeta(k)
		if err != nil {
			return err
		}
		tasks = append(tasks, meta)
		return nil
	}); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (ms *CarModelStore) GetModels(key types.TipSetKey, meta v2.ModelMeta) ([]v2.LilyModel, error) {
	models, err := ms.ModelMultiMapForTipSet(key)
	if err != nil {
		return nil, err
	}
	model, found, err := models.Get(ModelKeyer{meta})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no models found")
	}
	modelType := v2.ModelReflections[meta]

	// dear reader, I am sorry
	// the below code unmarshals the model into p and then copies p to newObj
	p := reflect.New(modelType.Type.Elem()).Interface().(v2.LilyModel)
	var out = make([]v2.LilyModel, 0, model.Length())
	if err := model.ForEach(p, func(i int64) error {
		newObj := reflect.New(reflect.TypeOf(p).Elem())
		oldVal := reflect.ValueOf(p).Elem()
		newVal := newObj.Elem()
		for i := 0; i < oldVal.NumField(); i++ {
			newValField := newVal.Field(i)
			if newValField.CanSet() {
				newValField.Set(oldVal.Field(i))
			}
		}
		out = append(out, newObj.Interface().(v2.LilyModel))
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (ms *CarModelStore) TipSets() []types.TipSetKey {
	out := make([]types.TipSetKey, 0, len(ms.tsModelMmap))
	for ts := range ms.tsModelMmap {
		out = append(out, ts)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out
}

func (ms *CarModelStore) Close() error {
	return ms.ro.Close()
}

func ParseTipSetString(ts string) ([]cid.Cid, error) {
	strs := strings.Split(ts, ",")

	var cids []cid.Cid
	for _, s := range strs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	return cids, nil
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
		if err := c.modelMap.Add(ModelKeyer{d.Meta()}, d); err != nil {
			return cid.Undef, err
		}
	}
	if err := c.modelMap.ForAll(func(k string, arr *adt.Array) error {
		r, err := arr.Root()
		if err != nil {
			return err
		}
		log.Infow("modelMap", "model", k, "root", r.String())
		return nil
	}); err != nil {
		return cid.Undef, err
	}
	modelRoot, err := c.modelMap.Root()
	if err != nil {
		return cid.Undef, err
	}
	if err = c.stateMap.Put(TipsetKeyer{ts.Key()}, typegen.CborCid(modelRoot)); err != nil {
		return cid.Undef, err
	}

	log.Infow("finalized model root", "root", modelRoot.String())
	return c.stateMap.Root()
}

func (c *modelStore) PutModel(ctx context.Context, m v2.LilyModel) error {
	c.cache = append(c.cache, m)
	return nil
}
