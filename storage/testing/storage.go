package testing

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/testutil"
)

var testDatabase = os.Getenv("LILY_TEST_DB")

func WaitForExclusiveMigratedStorage(ctx context.Context, tb testing.TB, debugLogs bool) (*storage.Database, func() error) {
	db, err := storage.NewDatabase(ctx, testDatabase, 10, tb.Name(), "public", false)
	require.NoError(tb, err)

	dbVersion, latest, err := db.GetSchemaVersions(ctx)
	require.NoError(tb, err)
	if dbVersion != latest {
		err = db.MigrateSchema(ctx)
		require.NoError(tb, err)
	}

	err = db.Connect(ctx)
	require.NoError(tb, err)

	release, err := testutil.WaitForExclusiveDatabaseLock(ctx, db.AsORM())
	if err != nil {
		db.Close(ctx) // nolint: errcheck
		tb.Fatalf("failed to get exclusive database access: %v", err)
	}

	cleanup := func() error {
		_ = release()
		return db.Close(ctx) // nolint: errcheck
	}

	if debugLogs {
		db.AsORM().AddQueryHook(&LoggingQueryHook{})
	}
	return db, cleanup
}

type LoggingQueryHook struct{}

func (l *LoggingQueryHook) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	q, err := event.FormattedQuery()
	if err != nil {
		return nil, err
	}

	if event.Err != nil {
		fmt.Printf("%s executing a query:\n%s\n", event.Err, q)
	}
	fmt.Println(string(q))

	return ctx, nil
}

func (l *LoggingQueryHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	return nil
}
