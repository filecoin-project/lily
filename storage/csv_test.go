package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/model"
)

type TestModel struct {
	Height  int64  `pg:",pk,notnull,use_zero"`
	Block   string `pg:",pk,notnull"`
	Message string `pg:",pk,notnull"`
}

func (tm *TestModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, tm)
}

type TimeModel struct {
	Height    int64     `pg:",pk,notnull,use_zero"`
	Processed time.Time `pg:",pk,notnull"`
}

func (tm *TimeModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, tm)
}

type InterfaceModel struct {
	Height int64 `pg:",pk,notnull,use_zero"`
	Value  interface{}
}

func (im *InterfaceModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, im)
}

type InterfaceJSONModel struct {
	Height int64       `pg:",pk,notnull,use_zero"`
	Value  interface{} `pg:",type:jsonb"`
}

func (im *InterfaceJSONModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, im)
}

type JSONModel struct {
	Height int64  `pg:",pk,notnull,use_zero"`
	Value  string `pg:",type:jsonb"` // this is a string that already contains json and should not be encoded again
}

func (tm *JSONModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, tm)
}

type StringSliceModel struct {
	Height    int64    `pg:",pk,notnull,use_zero"`
	Addresses []string // this will automatically be encoded as json
}

func (im *StringSliceModel) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	return s.PersistModel(ctx, im)
}

type ProcessingError struct {
	Source string
	Error  string
}

func TestCSVTable(t *testing.T) {
	tm := &TestModel{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	table := getCSVModelTable(tm, model.Version{Major: 1})
	require.NotNil(t, table.columns)
	assert.ElementsMatch(t, table.columns, []string{"height", "block", "message"})

	require.NotNil(t, table.fields)
	assert.ElementsMatch(t, table.fields, []string{"Height", "Block", "Message"})
}

func TestCSVPersist(t *testing.T) {
	tm := &TestModel{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "test_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,block,message\n"+
			"42,blocka,msg1\n",
		string(written))
}

func TestCSVPersistMulti(t *testing.T) {
	tms := []model.Persistable{
		&TestModel{
			Height:  42,
			Block:   "blocka",
			Message: "msg1",
		},

		&TestModel{
			Height:  43,
			Block:   "blockb",
			Message: "msg2",
		},

		&TestModel{
			Height:  44,
			Block:   "blockc",
			Message: "msg3",
		},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tms...)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "test_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,block,message\n"+
			"42,blocka,msg1\n"+
			"43,blockb,msg2\n"+
			"44,blockc,msg3\n",
		string(written))
}

type OtherTestModel struct {
	Height int64 `pg:",pk,notnull,use_zero"`
}

func (otm *OtherTestModel) Persist(ctx context.Context, tx *pg.Tx) error {
	return nil
}

type Composite struct {
	Thing *TestModel
	Other *OtherTestModel
}

func (c *Composite) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if err := s.PersistModel(ctx, c.Thing); err != nil {
		return err
	}
	if err := s.PersistModel(ctx, c.Other); err != nil {
		return err
	}
	return nil
}

func TestCSVPersistComposite(t *testing.T) {
	// Composite is a Marshaler so it can specify how it should marshal its fields
	comp := &Composite{
		Thing: &TestModel{
			Height:  42,
			Block:   "blocka",
			Message: "msg1",
		},

		Other: &OtherTestModel{
			Height: 42,
		},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), comp)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "test_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,block,message\n"+
			"42,blocka,msg1\n",
		string(written))

	otherWritten, err := ioutil.ReadFile(filepath.Join(dir, "other_test_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height\n"+
			"42\n",
		string(otherWritten))
}

func TestCSVPersistTime(t *testing.T) {
	// use time.Now since the default string value includes the monotonic clock, so we can test it is not present in csv output
	now := time.Now()

	tm := &TimeModel{
		Height:    42,
		Processed: now,
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "time_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,processed\n"+
			"42,"+now.Format(PostgresTimestampFormat)+"\n",
		string(written))
}

func TestCSVPersistInterfaceNil(t *testing.T) {
	tm := &InterfaceModel{
		Height: 42,
		Value:  nil,
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "interface_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,NULL\n",
		string(written))
}

func TestCSVPersistInterfaceValue(t *testing.T) {
	tm := &InterfaceModel{
		Height: 42,
		Value: []*ProcessingError{
			{
				Source: "some task",
				Error:  "processing error",
			},
		},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "interface_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,\"[{\"\"Source\"\":\"\"some task\"\",\"\"Error\"\":\"\"processing error\"\"}]\"\n",
		string(written))
}

func TestCSVPersistInterfaceNilJSON(t *testing.T) {
	tm := &InterfaceJSONModel{
		Height: 42,
		Value:  nil,
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "interface_json_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,null\n",
		string(written))
}

func TestCSVPersistInterfaceValueJSON(t *testing.T) {
	tm := &InterfaceJSONModel{
		Height: 42,
		Value:  []string{"f083047", "f088207"},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "interface_json_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,"+`"[""f083047"",""f088207""]"`+"\n",
		string(written))
}

func TestCSVPersistValueJSON(t *testing.T) {
	tm := &JSONModel{
		Height: 42,
		Value:  `{"some":"json"}`,
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "json_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,"+`"{""some"":""json""}"`+"\n",
		string(written))
}

func TestCSVPersistValueStringSlice(t *testing.T) {
	tm := &StringSliceModel{
		Height:    42,
		Addresses: []string{"f083047", "f088207"},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir) // nolint: errcheck

	st, err := NewCSVStorage(dir, model.Version{Major: 1}, DefaultCSVStorageOptions())
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "string_slice_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,addresses\n"+
			"42,"+`"[""f083047"",""f088207""]"`+"\n",
		string(written))
}

type VersionedModelLatest struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"versioned_model"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	Block     string   `pg:",notnull"`
	Message   string   `pg:",notnull"`
}

// VersionedModelV2 is an older version of VersionedModel that uses same table name but different structure
type VersionedModelV2 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"versioned_model"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	Block     string   `pg:",notnull"`
}

func (vm *VersionedModelLatest) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	switch version {
	case model.Version{Major: 3}:
		return s.PersistModel(ctx, vm)

	case model.Version{Major: 2}:
		v1 := &VersionedModelV2{
			Height: vm.Height,
			Block:  vm.Block,
		}
		return s.PersistModel(ctx, v1)
	case model.Version{Major: 1}:
		// Model did not exist in schema version 1, so don't attempt to persist
		return nil
	default:
		return fmt.Errorf("Unsupported version: %s", version)
	}
}

func TestCSVTableWithVersion(t *testing.T) {
	vm := &VersionedModelLatest{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	table := getCSVModelTable(vm, model.Version{Major: 3})
	require.NotNil(t, table.columns)
	assert.ElementsMatch(t, table.columns, []string{"height", "block", "message"})

	require.NotNil(t, table.fields)
	assert.ElementsMatch(t, table.fields, []string{"Height", "Block", "Message"})

	vm1 := &VersionedModelV2{
		Height: 42,
		Block:  "blocka",
	}

	tablev1 := getCSVModelTable(vm1, model.Version{Major: 2})
	require.NotNil(t, tablev1.columns)
	assert.ElementsMatch(t, tablev1.columns, []string{"height", "block"})

	require.NotNil(t, tablev1.fields)
	assert.ElementsMatch(t, tablev1.fields, []string{"Height", "Block"})
}

func TestCSVPersistWithVersion(t *testing.T) {
	vm := &VersionedModelLatest{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	// Persist latest version
	t.Run("latest", func(t *testing.T) {
		dir, err := ioutil.TempDir("", strings.ReplaceAll(t.Name(), "/", "_"))
		require.NoError(t, err)

		defer os.RemoveAll(dir) // nolint: errcheck

		st, err := NewCSVStorage(dir, model.Version{Major: 3}, DefaultCSVStorageOptions())
		require.NoError(t, err)

		err = st.PersistBatch(context.Background(), vm)
		require.NoError(t, err)

		written, err := ioutil.ReadFile(filepath.Join(dir, "versioned_model.csv"))
		require.NoError(t, err)
		assert.EqualValues(t,
			"height,block,message\n"+
				"42,blocka,msg1\n",
			string(written))
	})

	// Persist version 2 of same model
	t.Run("v2", func(t *testing.T) {
		// Latest version
		dir, err := ioutil.TempDir("", strings.ReplaceAll(t.Name(), "/", "_"))
		require.NoError(t, err)

		defer os.RemoveAll(dir) // nolint: errcheck

		st, err := NewCSVStorage(dir, model.Version{Major: 2}, DefaultCSVStorageOptions())
		require.NoError(t, err)

		err = st.PersistBatch(context.Background(), vm)
		require.NoError(t, err)

		written, err := ioutil.ReadFile(filepath.Join(dir, "versioned_model.csv"))
		require.NoError(t, err)
		assert.EqualValues(t,
			"height,block\n"+
				"42,blocka\n",
			string(written))
	})
}

func TestCSVOptionOmitHeader(t *testing.T) {
	tm := &TestModel{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	baseName := t.Name()

	runTest := func(t *testing.T, omitHeader bool, expected string) {
		dir, err := ioutil.TempDir("", baseName)
		t.Logf("dir %s", dir)
		require.NoError(t, err)

		defer os.RemoveAll(dir) // nolint: errcheck

		opts := DefaultCSVStorageOptions()
		opts.OmitHeader = omitHeader

		st, err := NewCSVStorage(dir, model.Version{Major: 1}, opts)
		require.NoError(t, err)

		err = st.PersistBatch(context.Background(), tm)
		require.NoError(t, err)

		written, err := ioutil.ReadFile(filepath.Join(dir, "test_models.csv"))
		require.NoError(t, err)
		assert.EqualValues(t, expected, string(written))
	}

	t.Run("false", func(t *testing.T) {
		runTest(t, false, "height,block,message\n"+"42,blocka,msg1\n")
	})

	t.Run("true", func(t *testing.T) {
		runTest(t, true, "42,blocka,msg1\n")
	})
}

func TestCSVOptionFilePattern(t *testing.T) {
	tm := &TestModel{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	baseName := t.Name()

	runTest := func(t *testing.T, pattern string, md Metadata, expected string) {
		dir, err := ioutil.TempDir("", baseName)
		t.Logf("dir %s", dir)
		require.NoError(t, err)

		defer os.RemoveAll(dir) // nolint: errcheck

		opts := DefaultCSVStorageOptions()
		opts.FilePattern = pattern

		st, err := NewCSVStorage(dir, model.Version{Major: 1}, opts)
		require.NoError(t, err)

		st = st.WithMetadata(md)

		err = st.PersistBatch(context.Background(), tm)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(dir, expected))
		require.NoError(t, err)
	}

	t.Run("default", func(t *testing.T) {
		runTest(t, "", Metadata{}, "test_models.csv")
	})

	t.Run("table", func(t *testing.T) {
		runTest(t, "{table}.csv", Metadata{}, "test_models.csv")
	})

	t.Run("jobname", func(t *testing.T) {
		runTest(t, "{jobname}.csv", Metadata{JobName: "job1"}, "job1.csv")
	})
}
