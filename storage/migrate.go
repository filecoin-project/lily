package storage

import (
	"context"
	"strconv"

	_ "github.com/filecoin-project/sentinel-visor/storage/migrations"
	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
)

// GetSchemaVersions returns the schema version in the database and the latest schema version defined by the available
// migrations.
func (d *Database) GetSchemaVersions(ctx context.Context) (model.Version, model.Version, error) {
	// If we're already connected then use that connection
	if d.DB != nil {
		return getSchemaVersions(ctx, d.DB)
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("connect: %w", err)
	}
	defer db.Close() // nolint: errcheck
	return getSchemaVersions(ctx, db)
}

// getSchemaVersions returns the schema version in the database and the schema version defined by the available
// migrations.
func getSchemaVersions(ctx context.Context, db *pg.DB) (model.Version, model.Version, error) {
	// Ensure the visor_version table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS public.visor_version (
			"major" int NOT NULL,
			PRIMARY KEY ("major")
		)
	`)
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("ensure visor_version exists :%w", err)
	}

	var major int
	_, err = db.QueryOne(pg.Scan(&major), `SELECT major FROM visor_version LIMIT 1`)
	if err != nil && err != pg.ErrNoRows {
		return model.Version{}, model.Version{}, err
	}

	// Run the migration init to ensure we always have a migrations table
	_, _, err = migrations.Run(db, "init")
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("migration table init: %w", err)
	}

	migration, err := migrations.Version(db)
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("unable to determine schema version: %w", err)
	}

	dbVersion := model.Version{
		Major: major,
		Patch: int(migration),
	}

	return dbVersion, LatestSchemaVersion(), nil
}

// LatestSchemaVersion returns the most recent version of the model schema. It is based on the highest migration version
// in the highest major schema version
func LatestSchemaVersion() model.Version {
	// TODO: detect major version
	return model.Version{Major: 0, Patch: getHighestMigration(migrations.DefaultCollection)}
}

func getHighestMigration(coll *migrations.Collection) int {
	var latestMigration int64
	ms := coll.Migrations()
	for _, m := range ms {
		if m.Version > latestMigration {
			latestMigration = m.Version
		}
	}
	return int(latestMigration)
}

// MigrateSchema migrates the database schema to the latest version based on the list of migrations available
func (d *Database) MigrateSchema(ctx context.Context) error {
	return d.MigrateSchemaTo(ctx, LatestSchemaVersion())
}

// MigrateSchema migrates the database schema to a specific version. Note that downgrading a schema to an earlier
// version is destructive and may result in the loss of data.
func (d *Database) MigrateSchemaTo(ctx context.Context, target model.Version) error {
	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}
	defer db.Close() // nolint: errcheck

	dbVersion, latestVersion, err := getSchemaVersions(ctx, db)
	if err != nil {
		return xerrors.Errorf("get schema versions: %w", err)
	}
	log.Infof("current database schema is version %s", dbVersion)

	if target.Major != dbVersion.Major {
		return xerrors.Errorf("cannot migrate to a different major schema version. database version=%s, target version=%s", dbVersion, target)
	}

	if latestVersion.Patch < target.Patch {
		return xerrors.Errorf("no migrations found for version %d", target)
	}

	if dbVersion == target {
		return xerrors.Errorf("database schema is already at version %d", dbVersion)
	}

	coll, err := collectionForVersion(target)
	if err != nil {
		return xerrors.Errorf("no schema definition corresponds to version %s", target)
	}

	if err := checkMigrationSequence(ctx, coll, dbVersion.Patch, target.Patch); err != nil {
		return xerrors.Errorf("check migration sequence: %w", err)
	}

	// Acquire an exclusive lock on the schema so we know no other instances are running
	if err := SchemaLock.LockExclusive(ctx, db); err != nil {
		return xerrors.Errorf("acquiring schema lock: %w", err)
	}

	// CREATE EXTENSION cannot run inside a transaction (as part of
	// migration), so we run it explicitally here and only when setting
	// things up for the first time (#179).
	if (dbVersion == model.Version{Major: 0, Patch: 0}) {
		if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public`); err != nil {
			return xerrors.Errorf("creating timescaledb extension: %w", err)
		}
	}

	// Remember to release the lock
	defer func() {
		err := SchemaLock.UnlockExclusive(ctx, db)
		if err != nil {
			log.Errorf("failed to release exclusive lock: %v", err)
		}
	}()

	// Do we need to rollback schema version
	if dbVersion.Patch > target.Patch {
		for dbVersion.Patch > target.Patch {
			log.Warnf("running destructive schema migration from patch %d to patch %d", dbVersion.Patch, dbVersion.Patch-1)
			_, newDBPatch, err := migrations.Run(db, "down")
			if err != nil {
				return xerrors.Errorf("run migration: %w", err)
			}
			dbVersion.Patch = int(newDBPatch)
			log.Infof("current database schema is now version %s", dbVersion)
		}
		return nil
	}

	// Need to advance schema version
	log.Infof("running schema migration from version %s to version %s", dbVersion, target)
	_, newDBPatch, err := migrations.Run(db, "up", strconv.Itoa(target.Patch))
	if err != nil {
		return xerrors.Errorf("run migration: %w", err)
	}

	dbVersion.Patch = int(newDBPatch)

	log.Infof("current database schema is now version %s", dbVersion)

	return nil
}

func checkMigrationSequence(ctx context.Context, coll *migrations.Collection, from, to int) error {
	versions := map[int64]bool{}
	ms := coll.Migrations()
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

func collectionForVersion(version model.Version) (*migrations.Collection, error) {
	if version.Major != 0 {
		panic("unsupported major version")
	}
	// TODO: detect major version
	return migrations.DefaultCollection, nil
}
