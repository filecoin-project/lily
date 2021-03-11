package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/sentinel-visor/model/actors/miner"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/filecoin-project/sentinel-visor/storage/migrations"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

func TestConsistentSchemaMigrationSequence(t *testing.T) {
	latestVersion := getLatestSchemaVersion()
	err := checkMigrationSequence(context.Background(), 1, latestVersion)
	require.NoError(t, err)
}

func TestSchemaIsCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	for _, m := range models {
		model := m
		t.Run(fmt.Sprintf("%T", model), func(t *testing.T) {
			q := db.Model(model)
			err := verifyModel(ctx, db, q.TableModel().Table())
			if err != nil {
				t.Errorf("%v", err)
				ctq := orm.NewCreateTableQuery(q, &orm.CreateTableOptions{IfNotExists: true})
				t.Logf("Expect %s", ctq.String())
			}
		})
	}
}

func TestModelUpsert(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	_, err = db.Exec(`TRUNCATE TABLE miner_infos`)
	require.NoError(t, err, "truncating miner_infos")

	// database disallowing upserting
	d := &Database{
		DB:     db,
		Clock:  testutil.NewMockClock(),
		Upsert: false,
	}

	// model was picked for this test since it has nullable fields and untagged pg fields.
	minerInfo := &miner.MinerInfo{
		Height:                  1,
		MinerID:                 "minerID",
		StateRoot:               "stateroot",
		OwnerID:                 "owner",
		WorkerID:                "worker",
		WorkerChangeEpoch:       0,
		ConsensusFaultedElapsed: 0,
		PeerID:                  "",
		ControlAddresses:        nil,
		MultiAddresses:          nil,
	}

	// the second insert should be ignored.
	err = d.PersistBatch(ctx, minerInfo)
	require.NoErrorf(t, err, "persisting miner info model: %v", err)
	err = d.PersistBatch(ctx, minerInfo)
	require.NoErrorf(t, err, "persisting miner info model: %v", err)

	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM miner_infos`)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count = 0
	// modify the database to permit upserting
	d.Upsert = true

	// modify the model, expect this change to persist after the upsert.
	minerInfo.OwnerID = "UPSERT"
	err = d.PersistBatch(ctx, minerInfo)
	require.NoErrorf(t, err, "persisting miner_info model: %v", err)

	// reset count, there should still be a single item in the table
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM miner_infos`)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var owner string
	_, err = db.QueryOne(pg.Scan(&owner), `SELECT owner_id FROM miner_infos`)
	require.NoError(t, err)
	assert.Equal(t, "UPSERT", owner)

}

func TestLongNames(t *testing.T) {
	justLongEnough := strings.Repeat("x", MaxPostgresNameLength)
	_, err := NewDatabase(context.Background(), "postgres://example.com/fakedb", 1, justLongEnough, false)
	require.NoError(t, err)

	tooLong := strings.Repeat("x", MaxPostgresNameLength+1)
	_, err = NewDatabase(context.Background(), "postgres://example.com/fakedb", 1, tooLong, false)
	require.Error(t, err)
}

// TestingUpsertStruct is only used for validating the GenerateUpsertStrings() method
type TestingUpsertStruct struct {
	// should be ignored by upsert generator
	tableName struct{} `pg:"testing_upsert_struct"` // nolint: structcheck,unused
	Ignored   string   `pg:"-"`

	// should be a constrained field in the conflict statement
	Height    int64  `pg:",pk,use_zero,notnull"`
	Cid       string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	// should be an unconstrained field in the upsert statement
	Heads     string `pg:",notnull"`
	Shoulders string `pg:",nopk"`
	Knees     uint64 `pg:",use_zero"`

	// currently we drop the `pg` tag from fields we allow as null, this is probably a bad habit.
	Toes      []byte
	CamelCase string
}

func (t *TestingUpsertStruct) ExpectedConflictStatement() string {
	return "(cid, height, state_root) DO UPDATE"
}

func (t *TestingUpsertStruct) ExpectedUpsertStatement() string {
	return `"camel_case" = EXCLUDED.camel_case, "heads" = EXCLUDED.heads, "knees" = EXCLUDED.knees, "shoulders" = EXCLUDED.shoulders, "toes" = EXCLUDED.toes`
}

func TestUpsertSQLGeneration(t *testing.T) {
	testModel := &TestingUpsertStruct{
		Ignored:   "ignored",
		Height:    1,
		Cid:       "cid",
		StateRoot: "stateroot",
		Heads:     "heads",
		Shoulders: "shoulders",
		Knees:     1,
		Toes:      []byte{1, 2, 3},
	}
	conflict, upsert := GenerateUpsertStrings(testModel)

	assert.Equal(t, testModel.ExpectedConflictStatement(), conflict)
	assert.Equal(t, testModel.ExpectedUpsertStatement(), upsert)
}
