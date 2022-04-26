package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/wait"
)

var testDatabase = os.Getenv("LILY_TEST_DB")

// DatabaseAvailable reports whether a database is available for testing
func DatabaseAvailable() bool {
	return testDatabase != ""
}

// Database returns the connection string for connecting to the test database
func DatabaseOptions() string {
	return testDatabase
}

// WaitForExclusiveDatabase waits for exclusive access to the test database until the context is done or the
// exclusive access is granted. It returns a cleanup function that should be called to close the database connection.
func WaitForExclusiveDatabase(ctx context.Context, tb testing.TB) (*pg.DB, func() error, error) {
	require.NotEmpty(tb, testDatabase, "No test database available: LILY_TEST_DB not set")
	opt, err := pg.ParseURL(testDatabase)
	require.NoError(tb, err)

	db := pg.Connect(opt)
	db = db.WithContext(ctx)

	// Check if connection credentials are valid and PostgreSQL is up and running.
	if err := db.Ping(ctx); err != nil {
		return nil, db.Close, xerrors.Errorf("ping database: %w", err)
	}

	release, err := WaitForExclusiveDatabaseLock(ctx, db)
	if err != nil {
		db.Close() // nolint: errcheck
		tb.Fatalf("failed to get exclusive database access: %v", err)
	}

	cleanup := func() error {
		_ = release()
		return db.Close() // nolint: errcheck
	}

	return db, cleanup, nil
}

const (
	testDatabaseLockID            = 88899888
	testDatabaseLockCheckInterval = 2 * time.Millisecond
)

// WaitForExclusiveDatabaseLock waits for a an exclusive lock on the test database until the context is done or the
// exclusive access is granted. It returns a cleanup function that should be called to release the exclusive lock. In any
// case the lock will be automatically released when the database session ends.
func WaitForExclusiveDatabaseLock(ctx context.Context, db *pg.DB) (func() error, error) {
	err := wait.RepeatUntil(ctx, testDatabaseLockCheckInterval, tryTestDatabaseLock(ctx, db))
	if err != nil {
		return nil, err
	}

	release := func() error {
		var released bool
		_, err := db.QueryOneContext(ctx, pg.Scan(&released), `SELECT pg_advisory_unlock(?);`, int64(testDatabaseLockID))
		if err != nil {
			return xerrors.Errorf("unlocking exclusive lock: %w", err)
		}
		if !released {
			return xerrors.Errorf("exclusive lock not released")
		}
		return nil
	}

	return release, nil
}

func tryTestDatabaseLock(ctx context.Context, db *pg.DB) func(context.Context) (bool, error) {
	return func(context.Context) (bool, error) {
		var acquired bool
		_, err := db.QueryOneContext(ctx, pg.Scan(&acquired), `SELECT pg_try_advisory_lock(?);`, int64(testDatabaseLockID))
		return acquired, err
	}
}

// TruncateBlockTables ensures the indexing tables are empty
func TruncateBlockTables(tb testing.TB, db *pg.DB) error {
	_, err := db.Exec(`TRUNCATE TABLE block_headers`)
	require.NoError(tb, err, "block_headers")

	_, err = db.Exec(`TRUNCATE TABLE block_parents`)
	require.NoError(tb, err, "block_parents")

	_, err = db.Exec(`TRUNCATE TABLE drand_block_entries`)
	require.NoError(tb, err, "drand_block_entries")

	return nil
}
