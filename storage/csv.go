package storage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/go-pg/pg/v10/orm"

	"github.com/filecoin-project/sentinel-visor/model"
)

const PostgresTimestampFormat = "2006-01-02T15:04:05.999Z07:00"

var ErrMarshalUnsupportedType = errors.New("cannot marshal unsupported type")

var (
	// Cache of model schemas for csv storage
	csvModelTablesMu sync.Mutex
	csvModelTables   = map[string]table{}
)

func getCSVModelTable(v interface{}) table {
	q := orm.NewQuery(nil, v)
	tm := q.TableModel()
	m := tm.Table()
	name := stripQuotes(m.SQLNameForSelects)

	csvModelTablesMu.Lock()
	defer csvModelTablesMu.Unlock()

	t, ok := csvModelTables[name]
	if ok {
		return t
	}

	t.name = name
	for _, fld := range m.Fields {
		t.columns = append(t.columns, fld.SQLName)
		t.fields = append(t.fields, fld.GoName)
		t.types = append(t.types, fld.SQLType)
	}
	csvModelTables[name] = t

	return t
}

func getCSVModelTableByName(name string) (table, bool) {
	csvModelTablesMu.Lock()
	defer csvModelTablesMu.Unlock()

	t, ok := csvModelTables[name]
	return t, ok
}

type CSVStorage struct {
	path string
}

// A table is a list of columns and corresponding field names in the Go struct
type table struct {
	name    string
	columns []string
	fields  []string
	types   []string
}

func NewCSVStorage(path string) (*CSVStorage, error) {
	return &CSVStorage{
		path: path,
	}, nil
}

// PersistBatch persists a batch of models to CSV, creating new files if they don't already exist otherwise appending
// to existing ones.
func (c *CSVStorage) PersistBatch(ctx context.Context, ps ...model.Persistable) error {
	batch := &CSVBatch{
		data: map[string][][]string{},
	}

	for _, p := range ps {
		if err := p.Persist(ctx, batch); err != nil {
			return err
		}
	}

	for name, rows := range batch.data {
		if len(rows) == 0 {
			continue
		}
		t, ok := getCSVModelTableByName(name)
		if !ok {
			log.Errorf("unknown table name: %s", name)
			continue
		}
		filename := filepath.Join(c.path, name+".csv")
		var w *csv.Writer

		// Try to create the file
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			// Created file successfully
			defer f.Close()

			// Write the headers
			w = csv.NewWriter(f)
			if err := w.Write(t.columns); err != nil {
				log.Errorw("failed to write csv headers", "error", err, "filename", filename)
				continue
			}
		} else {
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) || !os.IsExist(pathErr) {
				return fmt.Errorf("create file %q: %w", filename, err)
			}

			// File exists, attempt to append
			f, err = os.OpenFile(filename, os.O_APPEND|os.O_RDWR|os.O_EXCL, 0o644)
			if err != nil {
				return fmt.Errorf("open file %q: %w", filename, err)
			}
			defer f.Close()
			w = csv.NewWriter(f)
		}

		if err := w.WriteAll(rows); err != nil {
			log.Errorw("failed to write csv data", "error", err, "filename", filename)
			continue
		}

		w.Flush()
		if err := f.Sync(); err != nil {
			log.Errorw("failed to sync csv file", "error", err, "filename", filename)
		}
	}

	return nil
}

type CSVBatch struct {
	data map[string][][]string
}

func (c *CSVBatch) PersistModel(ctx context.Context, m interface{}) error {
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
			if err := c.PersistModel(ctx, value.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		// Get the table for this type
		t := getCSVModelTable(m)

		// Build the row
		row := make([]string, len(t.fields))
		for i, f := range t.fields {
			fv := value.FieldByName(f)
			fk := fv.Kind()
			if (fk == reflect.Slice || fk == reflect.Map || fk == reflect.Ptr || fk == reflect.Chan || fk == reflect.Func || fk == reflect.Interface) && fv.IsNil() {
				switch t.types[i] {
				case "json", "jsonb":
					row[i] = "null" // this is a json value of null
				default:
					row[i] = "NULL"
				}
				continue
			}

			ft := fv.Type()

			// Special formatting for known types
			if ft.PkgPath() == "time" && ft.Name() == "Time" {
				v := fv.Interface().(time.Time)
				row[i] = v.Format(PostgresTimestampFormat)
				continue
			}

			var encodeAsJSON bool

			// Strings marked as json type are assumed to already be encoded
			if fk != reflect.String && (t.types[i] == "json" || t.types[i] == "jsonb") {
				encodeAsJSON = true
			} else if fk == reflect.Interface {
				encodeAsJSON = true
			}

			if encodeAsJSON {
				v, err := json.Marshal(fv.Interface())
				if err != nil {
					return err
				}
				row[i] = string(v)
				continue
			}

			row[i] = fmt.Sprint(fv)
		}
		c.data[t.name] = append(c.data[t.name], row)
		return nil
	default:
		return ErrMarshalUnsupportedType

	}
}
