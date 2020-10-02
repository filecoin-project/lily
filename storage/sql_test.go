package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

func init() {
	// Freeze time for tests
	timeNow = testutil.KnownTimeNow
}

func TestSchemaIsCurrent(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

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
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedTipsets := visor.ProcessingStateChangeList{
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
			TipSet:       "cid4",
			Height:       4,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 15),
			CompletedAt:  testutil.KnownTime.Add(-time.Minute * 5),
		},

		// TipSet claimed by another process that has expired
		{
			TipSet:       "cid5",
			Height:       5,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 5),
		},

		// TipSet claimed by another process
		{
			TipSet:       "cid6a,cid6b",
			Height:       6,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(time.Minute * 15),
		},
	}

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedTipsets.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed blocks: %v", err)
	}

	const batchSize = 3

	claimUntil := testutil.KnownTime.Add(time.Minute * 10)
	d := &Database{DB: db}

	claimed, err := d.LeaseStateChanges(ctx, claimUntil, batchSize, 500)
	require.NoError(t, err)
	require.Equal(t, batchSize, len(claimed), "number of claimed blocks")

	// TipSets are selected in descending height order, ignoring completed and claimed tipset
	assert.Equal(t, "cid5", claimed[0].TipSet, "first claimed tipset")
	assert.Equal(t, "cid3a", claimed[1].TipSet, "second claimed tipset")
	assert.Equal(t, "cid2a,cid2b,cid2c", claimed[2].TipSet, "third claimed tipset")

	// Check the database contains the leases
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_statechanges WHERE claimed_until=?`, claimUntil)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count)
}

func TestMarkStateChangeComplete(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedBlocks := visor.ProcessingStateChangeList{
		{
			TipSet:  "cid0",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},

		{
			TipSet:  "cid1a,cid1b",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},

		{
			TipSet:  "cid2",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},
	}

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedBlocks.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed blocks: %v", err)
	}

	d := &Database{DB: db}

	t.Run("with error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 1)
		err = d.MarkStateChangeComplete(ctx, "cid1a,cid1b", 1, completedAt, "message")
		require.NoError(t, err)

		// Check the database contains the updated row
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_statechanges WHERE completed_at=?`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("without error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 2)
		err = d.MarkStateChangeComplete(ctx, "cid1a,cid1b", 1, completedAt, "")
		require.NoError(t, err)

		// Check the database contains the updated row with a null errors_detected column
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_statechanges WHERE completed_at=? AND errors_detected IS NULL`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

// truncateVisorProcessingTables ensures the processing tables are empty
func truncateVisorProcessingTables(tb testing.TB, db *pg.DB) {
	tb.Helper()
	_, err := db.Exec(`TRUNCATE TABLE visor_processing_statechanges`)
	require.NoError(tb, err, "truncating visor_processing_statechanges")

	_, err = db.Exec(`TRUNCATE TABLE visor_processing_actors`)
	require.NoError(tb, err, "truncating visor_processing_actors")

	_, err = db.Exec(`TRUNCATE TABLE visor_processing_messages`)
	require.NoError(tb, err, "truncating visor_processing_messages")
}

func TestLeaseActors(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedActors := visor.ProcessingActorList{
		// Unclaimed, incomplete actor
		{
			Head:    "head0",
			Code:    "codeA",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete actor
		{
			Head:    "head1",
			Code:    "codeB",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete actor
		{
			Head:    "head2",
			Code:    "codeC",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},

		// Unclaimed, incomplete actor
		{
			Head:    "head3",
			Code:    "codeA",
			Height:  3,
			AddedAt: testutil.KnownTime,
		},

		// Actor completed with stale claim
		{
			Head:         "head4",
			Code:         "codeA",
			Height:       4,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 15),
			CompletedAt:  testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Actor claimed by another process that has expired
		{
			Head:         "head5",
			Code:         "codeA",
			Height:       5,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Actor claimed by another process
		{
			Head:         "head6",
			Code:         "codeA",
			Height:       6,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(time.Minute * 15),
		},
	}

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedActors.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed actors: %v", err)
	}

	const batchSize = 3
	allowedCodes := []string{"codeA", "codeB"}

	claimUntil := testutil.KnownTime.Add(time.Minute * 10)

	d := &Database{DB: db}
	claimed, err := d.LeaseActors(ctx, claimUntil, batchSize, 500, allowedCodes)
	require.NoError(t, err)
	require.Equal(t, batchSize, len(claimed), "number of claimed actors")

	// Blocks are selected in descending height order, ignoring completed and claimed blocks and only those will
	// allowed codes.
	assert.Equal(t, "head5", claimed[0].Head, "first claimed actor")
	assert.Equal(t, "head3", claimed[1].Head, "second claimed actor")
	assert.Equal(t, "head1", claimed[2].Head, "third claimed actor")

	// Check the database contains the leases
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_actors WHERE claimed_until=?`, claimUntil)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count)
}

func TestMarkActorComplete(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedActors := visor.ProcessingActorList{
		{
			Head:    "head0",
			Code:    "codeA",
			Height:  0,
			AddedAt: testutil.KnownTime,
		},
		{
			Head:    "head1",
			Code:    "codeB",
			Height:  1,
			AddedAt: testutil.KnownTime,
		},
		{
			Head:    "head2",
			Code:    "codeC",
			Height:  2,
			AddedAt: testutil.KnownTime,
		},
	}

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedActors.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed actors: %v", err)
	}

	d := &Database{DB: db}

	t.Run("with error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 1)
		err = d.MarkActorComplete(ctx, "head1", "codeB", completedAt, "message")
		require.NoError(t, err)

		// Check the database contains the updated row
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_actors WHERE completed_at=?`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("without error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 2)
		err = d.MarkActorComplete(ctx, "head1", "codeB", completedAt, "")
		require.NoError(t, err)

		// Check the database contains the updated row with a null errors_detected column
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_actors WHERE completed_at=? AND errors_detected IS NULL`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestLeaseBlockMessages(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedMessageTipSets := visor.ProcessingMessageList{
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
			TipSet:       "cid4a",
			Height:       4,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 15),
			CompletedAt:  testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Tipset claimed by another process that has expired
		{
			TipSet:       "cid5a",
			Height:       5,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(-time.Minute * 5),
		},

		// Tipset claimed by another process
		{
			TipSet:       "cid6a,cid6b",
			Height:       6,
			AddedAt:      testutil.KnownTime,
			ClaimedUntil: testutil.KnownTime.Add(time.Minute * 15),
		},
	}

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedMessageTipSets.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed blocks: %v", err)
	}

	const batchSize = 3

	claimUntil := testutil.KnownTime.Add(time.Minute * 10)
	d := &Database{DB: db}

	claimed, err := d.LeaseTipSetMessages(ctx, claimUntil, batchSize, 500)
	require.NoError(t, err)
	require.Equal(t, batchSize, len(claimed), "number of claimed message blocks")

	// Blocks are selected in descending height order, ignoring completed and claimed blocks
	assert.Equal(t, "cid5a", claimed[0].TipSet, "first claimed message tipset")
	assert.Equal(t, "cid3a", claimed[1].TipSet, "second claimed message tipset")
	assert.Equal(t, "cid2a,cid2b,cid2c", claimed[2].TipSet, "third claimed message tipset")

	// Check the database contains the leases
	var count int
	_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_messages WHERE claimed_until=?`, claimUntil)
	require.NoError(t, err)
	assert.Equal(t, batchSize, count)
}

func TestMarkTipSetMessagesComplete(t *testing.T) {
	if testing.Short() || !testutil.DatabaseAvailable() {
		t.Skip("short testing requested or VISOR_TEST_DB not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	truncateVisorProcessingTables(t, db)

	indexedMessages := visor.ProcessingMessageList{
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

	if err := db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return indexedMessages.PersistWithTx(ctx, tx)
	}); err != nil {
		t.Fatalf("persisting indexed message blocks: %v", err)
	}

	d := &Database{DB: db}

	t.Run("with error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 1)
		err = d.MarkTipSetMessagesComplete(ctx, "cid1", 1, completedAt, "message")
		require.NoError(t, err)

		// Check the database contains the updated row
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_messages WHERE completed_at=?`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("without error message", func(t *testing.T) {
		completedAt := testutil.KnownTime.Add(time.Minute * 2)
		err = d.MarkTipSetMessagesComplete(ctx, "cid1", 1, completedAt, "")
		require.NoError(t, err)

		// Check the database contains the updated row with a null errors_detected column
		var count int
		_, err = db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_messages WHERE completed_at=? AND errors_detected IS NULL`, completedAt)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

}
