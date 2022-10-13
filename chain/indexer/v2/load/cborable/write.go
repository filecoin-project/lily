package cborable

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-state-types/builtin/v8/util/adt"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"

	v2 "github.com/filecoin-project/lily/model/v2"
)

func NewModelWriter(store adt.Store, bitwidth int) (*ModelWriter, error) {
	mmm, err := NewMetaModelMap(store, bitwidth)
	if err != nil {
		return nil, err
	}
	return &ModelWriter{
		store:        store,
		cache:        make(map[v2.ModelMeta][]cacheValue),
		metaModelMap: mmm,
	}, nil
}

type ModelWriter struct {
	store        adt.Store
	cache        map[v2.ModelMeta][]cacheValue
	metaModelMap *MetaModelMap
}

func (w *ModelWriter) StageModel(ctx context.Context, m v2.LilyModel) error {
	w.cache[m.Meta()] = append(w.cache[m.Meta()], newCacheValue(m))
	return nil
}

func (w *ModelWriter) Finalize(ctx context.Context) (r cid.Cid, err error) {
	defer func() {
		log.Infow("finalized meta model map", "root", r.String())
	}()
	if len(w.cache) == 0 {
		return cid.Undef, fmt.Errorf("no models staged")
	}
	if err := w.persistCache(); err != nil {
		return cid.Undef, err
	}
	return w.metaModelMap.Root()
}

func (w *ModelWriter) sortCache() error {
	start := time.Now()
	defer func() {
		log.Infow("sorted model cache", "duration", time.Since(start))
	}()

	grp := errgroup.Group{}
	for meta, models := range w.cache {
		meta := meta
		models := models
		grp.Go(func() error {
			sort.Slice(models, func(i, j int) bool {
				return models[i].cid.KeyString() < models[j].cid.KeyString()
			})
			log.Infow("sort model array", "meta", meta.String(), "size", len(models))
			return nil
		})
	}
	return grp.Wait()
}

func (w *ModelWriter) persistCache() error {
	if err := w.sortCache(); err != nil {
		return err
	}
	start := time.Now()
	defer func() {
		log.Infow("wrote cache", "duration", time.Since(start))
	}()
	grp := errgroup.Group{}
	for k, v := range w.cache {
		meta := k
		models := v
		grp.Go(func() error {
			array, err := adt.MakeEmptyArray(w.store, BitWidth)
			if err != nil {
				return err
			}
			for _, model := range models {
				if err := array.AppendContinuous(model.model); err != nil {
					return err
				}
			}
			root, err := array.Root()
			if err != nil {
				return err
			}
			log.Infow("put model array", "meta", meta.String(), "root", root.String(), "size", array.Length())
			return w.metaModelMap.Put(meta, root)
		})
	}
	return grp.Wait()
}

func NewMetaModelMap(store adt.Store, bitwidth int) (*MetaModelMap, error) {
	m, err := adt.MakeEmptyMap(store, bitwidth)
	if err != nil {
		return nil, err
	}
	return &MetaModelMap{
		store:   store,
		metaMap: m,
	}, nil
}

type MetaModelMap struct {
	store   adt.Store
	metaMu  sync.Mutex
	metaMap *adt.Map
}

func (m *MetaModelMap) Put(meta v2.ModelMeta, root cid.Cid) error {
	m.metaMu.Lock()
	defer m.metaMu.Unlock()
	r := typegen.CborCid(root)
	return m.metaMap.Put(ModelKeyer{meta}, &r)
}

func (m *MetaModelMap) Root() (cid.Cid, error) {
	m.metaMu.Lock()
	defer m.metaMu.Unlock()
	return m.metaMap.Root()
}

// used to precalcuate model CID to avoid it being a bottleneck in sort since calling CID in the sort method will
// marshal cbor each time
func newCacheValue(m v2.LilyModel) cacheValue {
	return cacheValue{
		model: m,
		cid:   m.Cid(),
	}
}

type cacheValue struct {
	model v2.LilyModel
	cid   cid.Cid
}
