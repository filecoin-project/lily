package storage

import (
	"context"
	"reflect"
	"sync"

	"github.com/go-pg/pg/v10/orm"

	"github.com/filecoin-project/lily/model"
)

func NewMemStorage(version model.Version) *MemStorage {
	return &MemStorage{
		Data:    map[string][]interface{}{},
		Version: version,
	}
}

func NewMemStorageLatest() *MemStorage {
	return NewMemStorage(LatestSchemaVersion())
}

type MemStorage struct {
	// TODO parallel map?
	Data    map[string][]interface{}
	DataMu  sync.Mutex
	Version model.Version
}

func (j *MemStorage) PersistModel(ctx context.Context, m interface{}) error {
	if len(models) == 0 {
		return nil
	}

	value := reflect.ValueOf(m)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := j.PersistModel(ctx, value.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		q := orm.NewQuery(nil, m)
		tm := q.TableModel()
		n := tm.Table()
		name := stripQuotes(n.SQLNameForSelects)
		j.DataMu.Lock()
		j.Data[name] = append(j.Data[name], m)
		j.DataMu.Unlock()
		return nil
	default:
		return ErrMarshalUnsupportedType

	}
}

func (j *MemStorage) PersistBatch(ctx context.Context, ps ...model.Persistable) error {
	for _, p := range ps {
		if err := p.Persist(ctx, j, j.Version); err != nil {
			return err
		}
	}
	return nil
}
