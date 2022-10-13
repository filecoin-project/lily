package cborable

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sort"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipld/go-car"

	v2 "github.com/filecoin-project/lily/model/v2"
)

type ModelReader struct {
	modelMap map[types.TipSetKey]*adt.Multimap
	states   []ModelStateContainer
}

func NewModelReader(ctx context.Context, r io.Reader) (*ModelReader, error) {
	bs := blockstore.NewMemorySync()
	header, err := car.LoadCar(ctx, bs, r)
	if err != nil {
		return nil, fmt.Errorf("loading car: %w", err)
	}
	if len(header.Roots) != 1 {
		return nil, fmt.Errorf("invalud car header, expcted 1 root, got %d", len(header.Roots))
	}

	store := adt.WrapBlockStore(ctx, bs)
	stateMap, err := adt.AsMap(store, header.Roots[0], BitWidth)
	if err != nil {
		return nil, err
	}
	var stateContainer ModelStateContainer
	tsModelMultiMap := make(map[types.TipSetKey]*adt.Multimap)
	states := make([]ModelStateContainer, 0)
	if err := stateMap.ForEach(&stateContainer, func(key string) error {
		states = append(states, stateContainer)
		modelMmap, err := adt.AsMultimap(store, stateContainer.Models, BitWidth, BitWidth)
		if err != nil {
			return err
		}
		tsModelMultiMap[stateContainer.Current.Key()] = modelMmap
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].Current.Height() < states[j].Current.Height()
	})
	return &ModelReader{modelMap: tsModelMultiMap, states: states}, nil
}

func (r *ModelReader) GetModels(key types.TipSetKey, meta v2.ModelMeta) ([]v2.LilyModel, error) {
	models, err := r.modelMapForTipSet(key)
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

func (r *ModelReader) States() []ModelStateContainer {
	return r.states
}

func (r *ModelReader) ModelMetasForTipSet(key types.TipSetKey) ([]v2.ModelMeta, error) {
	models, err := r.modelMapForTipSet(key)
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

func (r *ModelReader) modelMapForTipSet(key types.TipSetKey) (*adt.Multimap, error) {
	models, found := r.modelMap[key]
	if !found {
		return nil, fmt.Errorf("no models for tipset %s", key)
	}
	return models, nil
}
