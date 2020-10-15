package storage

import (
	"context"
	"strconv"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	sm "github.com/filecoin-project/sentinel-visor/storage/migrations"
)

// GetSchemaVersions returns the schema version in the database and the latest schema version defined by the available
// migrations.
func (d *Database) GetSchemaVersions(ctx context.Context) (int, int, error) {
	// If we're already connected then use that connection
	if d.DB != nil {
		return getSchemaVersions(ctx, d.DB)
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return 0, 0, xerrors.Errorf("connect: %w", err)
	}
	defer db.Close()
	return getSchemaVersions(ctx, db)
}

// getSchemaVersions returns the schema version in the database and the schema version defined by the available
// migrations.
func getSchemaVersions(ctx context.Context, db *pg.DB) (int, int, error) {
	// Run the migration init to ensure we always have a migrations table
	_, _, err := migrations.Run(db, "init")
	if err != nil {
		return 0, 0, xerrors.Errorf("migration table init: %w", err)
	}

	dbVersion, err := migrations.Version(db)
	if err != nil {
		return 0, 0, xerrors.Errorf("unable to determine schema version: %w", err)
	}

	latestVersion := getLatestSchemaVersion()
	return int(dbVersion), latestVersion, nil
}

// Latest schema version is based on the highest migration version
func getLatestSchemaVersion() int {
	var latestVersion int64
	ms := migrations.DefaultCollection.Migrations()
	for _, m := range ms {
		if m.Version > latestVersion {
			latestVersion = m.Version
		}
	}
	return int(latestVersion)
}

// MigrateSchema migrates the database schema to the latest version based on the list of migrations available
func (d *Database) MigrateSchema(ctx context.Context) error {
	return d.MigrateSchemaTo(ctx, getLatestSchemaVersion())
}

// MigrateSchema migrates the database schema to a specific version. Note that downgrading a schema to an earlier
// version is destructive and may result in the loss of data.
func (d *Database) MigrateSchemaTo(ctx context.Context, target int) error {
	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}
	defer db.Close()

	dbVersion, latestVersion, err := getSchemaVersions(ctx, db)
	if err != nil {
		return xerrors.Errorf("get schema versions: %w", err)
	}
	log.Infof("current database schema is version %d", dbVersion)

	if latestVersion < target {
		return xerrors.Errorf("no migrations found for version %d", target)
	}

	if dbVersion == target {
		return xerrors.Errorf("database schema is already at version %d", dbVersion)
	}

	if err := checkMigrationSequence(ctx, dbVersion, target); err != nil {
		return xerrors.Errorf("check migration sequence: %w", err)
	}

	// Acquire an exclusive lock on the schema so we know no other instances are running
	if err := SchemaLock.LockExclusive(ctx, db); err != nil {
		return xerrors.Errorf("acquiring schema lock: %w", err)
	}

	// Remember to release the lock
	defer func() {
		err := SchemaLock.UnlockExclusive(ctx, db)
		if err != nil {
			log.Errorf("failed to release exclusive lock: %v", err)
		}
	}()

	// Do we need to rollback schema version
	if dbVersion > target {
		for dbVersion > target {
			log.Warnf("running destructive schema migration from version %d to version %d", dbVersion, dbVersion-1)
			_, newDBVersion, err := migrations.Run(db, "down")
			if err != nil {
				return xerrors.Errorf("run migration: %w", err)
			}
			dbVersion = int(newDBVersion)
			log.Infof("current database schema is now version %d", dbVersion)
		}
		return nil
	}

	// Need to advance schema version
	log.Infof("running schema migration from version %d to version %d", dbVersion, target)
	_, newDBVersion, err := migrations.Run(db, "up", strconv.Itoa(target))
	if err != nil {
		return xerrors.Errorf("run migration: %w", err)
	}
	log.Infof("current database schema is now version %d", newDBVersion)

	return nil
}

func checkMigrationSequence(ctx context.Context, from, to int) error {
	versions := map[int64]bool{}
	ms := migrations.DefaultCollection.Migrations()
	for _, m := range ms {
		if versions[m.Version] {
			return xerrors.Errorf("duplication migration for schema version %d", m.Version)
		}
		versions[m.Version] = true
	}

	if from > to {
		to, from = from, to
	}

	for i := from; i <= to; i++ {
		if !versions[int64(i)] {
			return xerrors.Errorf("missing migration for schema version %d", i)
		}
	}

	return nil
}

// GetDeferredSchemaVersions returns the version of the last deferred schema migration in the database and the
// highest available deferred migration less than or equal maxVersion
func (d *Database) GetDeferredSchemaVersions(ctx context.Context, maxVersion int) (int, int, error) {
	// If we're already connected then use that connection
	if d.DB != nil {
		return getDeferredSchemaVersions(ctx, d.DB, maxVersion)
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return 0, 0, xerrors.Errorf("connect: %w", err)
	}
	defer db.Close()
	return getDeferredSchemaVersions(ctx, db, maxVersion)
}

func getDeferredSchemaVersions(ctx context.Context, db *pg.DB, maxVersion int) (int, int, error) {
	if err := createDeferredSchemaTable(ctx, db); err != nil {
		return 0, 0, xerrors.Errorf("deferred migration table init: %w", err)
	}

	var dbVersion int64
	_, err := db.QueryOne(pg.Scan(&dbVersion), `SELECT version FROM public.visor_deferred_schema_migrations ORDER BY id DESC LIMIT 1`)
	if err != nil && err != pg.ErrNoRows {
		return 0, 0, xerrors.Errorf("query migration table: %w", err)
	}

	return int(dbVersion), getLatestDeferredSchemaVersion(maxVersion), nil
}

func createDeferredSchemaTable(ctx context.Context, db *pg.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS public.visor_deferred_schema_migrations (
			id serial,
			version bigint,
			created_at timestamptz
		)
	`)
	return err
}

func getLatestDeferredSchemaVersion(maxVersion int) int {
	var latestVersion int64
	dms := sm.DeferredMigrations()
	for _, m := range dms {
		if m.Version > latestVersion && m.Version <= int64(maxVersion) {
			latestVersion = m.Version
		}
	}
	return int(latestVersion)
}

func (d *Database) RunDeferredMigrations(ctx context.Context) error {
	dbVersion, _, err := d.GetSchemaVersions(ctx)
	if err != nil {
		return xerrors.Errorf("get schema versions: %w", err)
	}

	return d.RunDeferredMigrationsTo(ctx, dbVersion)
}

// RunDeferredMigrationsTo runs deferred migrations up to target version.
func (d *Database) RunDeferredMigrationsTo(ctx context.Context, target int) error {
	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}
	defer db.Close()

	dbVersion, latestVersion, err := getSchemaVersions(ctx, db)
	if err != nil {
		return xerrors.Errorf("get schema versions: %w", err)
	}
	if target > dbVersion {
		return xerrors.Errorf("target %d cannot be greater than current database schema %d", target, dbVersion)
	}
	if target > latestVersion {
		return xerrors.Errorf("target %d cannot be greater than latest available schema %d", target, latestVersion)
	}

	dbDeferredVersion, deferredLatestVersion, err := getDeferredSchemaVersions(ctx, db, target)
	if err != nil {
		return xerrors.Errorf("get deferred schema versions: %w", err)
	}

	if deferredLatestVersion > target {
		return xerrors.Errorf("latest deferred migration version is %d", deferredLatestVersion)
	}

	dms := sm.DeferredMigrations()

	// Do we need to rollback deferred migrations?
	if dbDeferredVersion > target {
		for dbDeferredVersion > target {
			dm, ok := dms[int64(dbDeferredVersion)]
			if !ok {
				log.Infof("skipping version %d, no deferred migration", dbDeferredVersion)
				dbDeferredVersion--
				continue
			}

			log.Warnf("running destructive schema migration from version %d to version %d", dbDeferredVersion, dbDeferredVersion-1)
			err := runDeferredMigration(ctx, db, dbDeferredVersion-1, dm.Down)
			if err != nil {
				return xerrors.Errorf("run migration: %w", err)
			}
			dbDeferredVersion--
			log.Infof("current deferred migration version is now %d", dbDeferredVersion)
		}
		return nil
	}

	// Need to apply deferred migrations
	for dbDeferredVersion < target {
		dm, ok := dms[int64(dbDeferredVersion)]
		if !ok {
			log.Infof("skipping version %d, no deferred migration", dbDeferredVersion)
			dbDeferredVersion++
			continue
		}

		log.Infof("running schema migration from version %d to version %d", dbDeferredVersion, dbDeferredVersion+1)
		err := runDeferredMigration(ctx, db, dbDeferredVersion, dm.Up) // note we pass current version
		if err != nil {
			return xerrors.Errorf("run migration: %w", err)
		}
		dbDeferredVersion++
		log.Infof("current deferred migration version is now %d", dbDeferredVersion)
	}
	return nil
}

func runDeferredMigration(ctx context.Context, db *pg.DB, newVersion int, fn func(migrations.DB) error) error {
	// Can't run in a transaction since deferred migrations often include concurrent index creation
	err := fn(db)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO public.visor_deferred_schema_migrations (version, created_at) VALUES (?,NOW())`, newVersion)
	if err != nil {
		return err
	}

	return nil
}
