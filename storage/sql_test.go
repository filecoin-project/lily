package storage

import (
	"context"
	"fmt"
	"github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"strings"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/model/visor"
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

	for _, model := range models {
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

func TestLeaseStateChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	truncateVisorProcessingTables(t, db)

	indexedTipsets := visor.ProcessingTipSetList{
		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid0a,cid0b",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid1a",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid2a,cid2b,cid2c",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid3a",
			Height:  3,
			AddedAt: testutil.KnownTime,
		},

		// TipSet completed with stale claim
		{
			TipSet:                  "cid4",
			Height:                  4,
			AddedAt:                 testutil.KnownTime,
			StatechangeClaimedUntil: testutil.KnownTime.Add(-time.Minute * 15),
			StatechangeCompletedAt:  testutil.KnownTime.Add(-time.Minute * 5),
		},

		// TipSet claimed by another process that has expired
		{
			TipSet:                  "cid5",
			Height:                  5,
			AddedAt:                 testutil.KnownTime,
			StatechangeClaimedUntil: testutil.KnownTime.Add(-time.Minute * 5),
		},

		// TipSet claimed by another process
		{
			TipSet:                  "cid6a,cid6b",
			Height:                  6,
			AddedAt:                 testutil.KnownTime,
			StatechangeClaimedUntil: testutil.KnownTime.Add(time.Minute * 15),
		},
	}

	d := &Database{
		DB:    db,
		Clock: testutil.NewMockClock(),
	}

	if err := d.PersistBatch(ctx, indexedTipsets); err != nil {
		t.Fatalf("persisting indexed blocks: %v", err)
	}

	const batchSize = 3

	claimUntil := testutil.KnownTime.Add(time.Minute * 10)

	claimed, err := d.LeaseStateChanges(ctx, claimUntil, batchSize, 0, 500)
	require.NoError(t, err)
	require.Equal(t, batchSize, len(claimed), "number of claimed blocks")

	// TipSets are selected in descending height order, ignoring completed and claimed tipset
	assert.Equal(t, "cid5", claimed[0].TipSet, "first claimed tipset")
	assert.Equal(t, "cid3a", claimed[1].TipSet, "second claimed tipset")
	assert.Equal(t, "cid2a,cid2b,cid2c", claimed[2].TipSet, "third claimed tipset")

	// Check the database contains the leases
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE statechange_claimed_until=?`, claimUntil)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count)
}

// truncateVisorProcessingTables ensures the processing tables are empty
func truncateVisorProcessingTables(tb testing.TB, db *pg.DB) {
	tb.Helper()
	_, err := db.Exec(`TRUNCATE TABLE visor_processing_tipsets`)
	require.NoError(tb, err, "truncating visor_processing_tipsets")

	_, err = db.Exec(`TRUNCATE TABLE visor_processing_actors`)
	require.NoError(tb, err, "truncating visor_processing_actors")

	_, err = db.Exec(`TRUNCATE TABLE visor_processing_messages`)
	require.NoError(tb, err, "truncating visor_processing_messages")

	_, err = db.Exec(`TRUNCATE TABLE messages`)
	require.NoError(tb, err, "truncating messages")

	_, err = db.Exec(`TRUNCATE TABLE receipts`)
	require.NoError(tb, err, "truncating receipts")

	_, err = db.Exec(`TRUNCATE TABLE block_headers`)
	require.NoError(tb, err, "truncating block_headers")

	_, err = db.Exec(`TRUNCATE TABLE block_messages`)
	require.NoError(tb, err, "truncating block_messages")
}

func TestLeaseTipSetEconomics(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	truncateVisorProcessingTables(t, db)

	indexedMessageTipSets := visor.ProcessingTipSetList{
		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid0a,cid0b",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid1a",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid2a,cid2b,cid2c",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete tipset
		{
			TipSet:  "cid3a",
			Height:  3,
			AddedAt: testutil.KnownTime,
		},

		// Tipset completed with stale claim
		{
			TipSet:                "cid4a",
			Height:                4,
			AddedAt:               testutil.KnownTime,
			EconomicsClaimedUntil: testutil.KnownTime.Add(-time.Minute * 15),
			EconomicsCompletedAt:  testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Tipset claimed by another process that has expired
		{
			TipSet:                "cid5a",
			Height:                5,
			AddedAt:               testutil.KnownTime,
			EconomicsClaimedUntil: testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Tipset claimed by another process
		{
			TipSet:                "cid6a,cid6b",
			Height:                6,
			AddedAt:               testutil.KnownTime,
			EconomicsClaimedUntil: testutil.KnownTime.Add(time.Minute * 15),
		},
	}

	d := &Database{
		DB:    db,
		Clock: testutil.NewMockClock(),
	}
	if err := d.PersistBatch(ctx, indexedMessageTipSets); err != nil {
		t.Fatalf("persisting indexed blocks: %v", err)
	}

	const batchSize = 3

	claimUntil := testutil.KnownTime.Add(time.Minute * 10)

	claimed, err := d.LeaseTipSetEconomics(ctx, claimUntil, batchSize, 0, 500)
	require.NoError(t, err)
	require.Equal(t, batchSize, len(claimed), "number of claimed message blocks")

	// Tipsets are selected in descending height order, ignoring completed and claimed blocks
	assert.Equal(t, "cid5a", claimed[0].TipSet, "first claimed message tipset")
	assert.Equal(t, "cid3a", claimed[1].TipSet, "second claimed message tipset")
	assert.Equal(t, "cid2a,cid2b,cid2c", claimed[2].TipSet, "third claimed message tipset")

	// Check the database contains the leases
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE economics_claimed_until=?`, claimUntil)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count)
}

func TestMarkTipSetComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	truncateVisorProcessingTables(t, db)

	indexedMessages := visor.ProcessingTipSetList{
		{
			TipSet:  "cid0a,cid0b",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},

		{
			TipSet:  "cid1",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},

		{
			TipSet:  "cid2",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},
	}

	d := &Database{
		DB:    db,
		Clock: testutil.NewMockClock(),
	}

	if err := d.PersistBatch(ctx, indexedMessages); err != nil {
		t.Fatalf("persisting indexed message blocks: %v", err)
	}

	t.Run("statechange with error", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 1)
		err = d.MarkStateChangeComplete(ctx, "cid1", 1, completedAt, "message")
		require.NoError(t, err)

		// Check the database contains the updated row
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE statechange_completed_at=?`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("statechange without error", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 2)
		err = d.MarkStateChangeComplete(ctx, "cid1", 1, completedAt, "")
		require.NoError(t, err)

		// Check the database contains the updated row with a null errors_detected column
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE statechange_completed_at=? AND statechange_errors_detected IS NULL`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("economics with error", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 1)
		err = d.MarkTipSetEconomicsComplete(ctx, "cid1", 1, completedAt, "message")
		require.NoError(t, err)

		// Check the database contains the updated row
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE economics_completed_at=?`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("economics without error", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 2)
		err = d.MarkTipSetEconomicsComplete(ctx, "cid1", 1, completedAt, "")
		require.NoError(t, err)

		// Check the database contains the updated row with a null errors_detected column
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets WHERE economics_completed_at=? AND economics_errors_detected IS NULL`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
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
