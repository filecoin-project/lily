package storage

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
