package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (tm *TestModel) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, tm)
}

type TimeModel struct {
	Height    int64     `pg:",pk,notnull,use_zero"`
	Processed time.Time `pg:",pk,notnull"`
}

func (tm *TimeModel) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, tm)
}

type InterfaceModel struct {
	Height int64 `pg:",pk,notnull,use_zero"`
	Value  interface{}
}

func (im *InterfaceModel) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, im)
}

type ProcessingError struct {
	Source string
	Error  error
}

func TestCSVTable(t *testing.T) {
	tm := &TestModel{
		Height:  42,
		Block:   "blocka",
		Message: "msg1",
	}

	table := getCSVModelTable(tm)
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

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
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

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
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

func (c *Composite) Persist(ctx context.Context, s model.StorageBatch) error {
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

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
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

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
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

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
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
				Error:  fmt.Errorf("processing error"),
			},
		},
	}

	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	st, err := NewCSVStorage(dir)
	require.NoError(t, err)

	err = st.PersistBatch(context.Background(), tm)
	require.NoError(t, err)

	written, err := ioutil.ReadFile(filepath.Join(dir, "interface_models.csv"))
	require.NoError(t, err)
	assert.EqualValues(t,
		"height,value\n"+
			"42,\"[{\"\"Source\"\":\"\"some task\"\",\"\"Error\"\":{}}]\"\n",
		string(written))
}
