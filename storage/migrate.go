package storage

import (
	"context"
	"strconv"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/schemas"
	v1 "github.com/filecoin-project/lily/schemas/v1"
)

// GetSchemaVersions returns the schema version in the database and the latest schema version defined by the available
// migrations.
func (d *Database) GetSchemaVersions(ctx context.Context) (model.Version, model.Version, error) {
	latest := LatestSchemaVersion()

	// If we're already connected then use that connection
	if d.db != nil {
		dbVersion, _, err := getDatabaseSchemaVersion(ctx, d.db, d.SchemaConfig())
		return dbVersion, latest, err
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("connect: %w", err)
	}
	defer db.Close() // nolint: errcheck
	dbVersion, _, err := getDatabaseSchemaVersion(ctx, db, d.SchemaConfig())
	return dbVersion, latest, err
}

// getDatabaseSchemaVersion returns the schema version in use by the database and whether the schema versioning
// tables have been initialized. If no schema version tables can be found then the database is assumed to be
// uninitialized and a zero version and false value will be returned. The returned boolean will only be true
// if the schema versioning tables exist and are populated correctly.
func getDatabaseSchemaVersion(ctx context.Context, db *pg.DB, cfg schemas.Config) (model.Version, bool, error) {
	vvExists, err := tableExists(ctx, db, cfg.SchemaName, "visor_version")
	if err != nil {
		return model.Version{}, false, xerrors.Errorf("checking if visor_version exists:%w", err)
	}

	migExists, err := tableExists(ctx, db, cfg.SchemaName, "gopg_migrations")
	if err != nil {
		return model.Version{}, false, xerrors.Errorf("checking if gopg_migrations exists:%w", err)
	}

	if !migExists && !vvExists {
		// Uninitialized database
		return model.Version{}, false, nil
	}

	// Ensure the visor_version table exists
	vvTableName := cfg.SchemaName + ".visor_version"
	var major int
	_, err = db.QueryOne(pg.Scan(&major), `SELECT major FROM ? LIMIT 1`, pg.SafeQuery(vvTableName))
	if err != nil && err != pg.ErrNoRows {
		return model.Version{}, false, err
	}

	coll, err := collectionForVersion(model.Version{
		Major: major,
	}, cfg)
	if err != nil {
		return model.Version{}, false, err
	}

	migration, err := coll.Version(db)
	if err != nil {
		return model.Version{}, false, xerrors.Errorf("unable to determine schema version: %w", err)
	}

	if major == 0 && migration == 0 {
		// Database has the version tables but they are unpopulated so database is not initialized
		return model.Version{}, false, nil
	}

	dbVersion := model.Version{
		Major: major,
		Patch: int(migration),
	}

	return dbVersion, true, nil
}

// initDatabaseSchema initializes the version tables for tracking schema version installed in the database
func initDatabaseSchema(ctx context.Context, db *pg.DB, cfg schemas.Config) error {
	if cfg.SchemaName != "public" {
		_, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS ?`, pg.SafeQuery(cfg.SchemaName))
		if err != nil {
			return xerrors.Errorf("ensure schema exists :%w", err)
		}
	}

	// Ensure the visor_version table exists
	vvTableName := cfg.SchemaName + ".visor_version"
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ? (
			"major" int NOT NULL,
			PRIMARY KEY ("major")
		)
	`, pg.SafeQuery(vvTableName))
	if err != nil {
		return xerrors.Errorf("ensure visor_version exists :%w", err)
	}

	// Ensure the gopg migrations table exists
	migTableName := cfg.SchemaName + ".gopg_migrations"
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ? (
			id serial,
			version bigint,
			created_at timestamptz
		)
	`, pg.SafeQuery(migTableName))
	if err != nil {
		return xerrors.Errorf("ensure visor_version exists :%w", err)
	}

	return nil
}

func validateDatabaseSchemaVersion(ctx context.Context, db *pg.DB, cfg schemas.Config) (model.Version, error) {
	// Check if the version of the schema is compatible
	dbVersion, initialized, err := getDatabaseSchemaVersion(ctx, db, cfg)
	if err != nil {
		return model.Version{}, xerrors.Errorf("get schema version: %w", err)
	}

	if !initialized {
		return model.Version{}, xerrors.Errorf("schema not installed in database")
	}

	if dbVersion.Before(LatestSchemaVersion()) {
		// the latest schema version supported by lily (LatestSchemaVersion()) is newer than the schema version in use by the database (`dbVersion`)
		// running lily this way will cause models data persistence failures.
		return model.Version{}, xerrors.Errorf("the latest schema version supported by lily (%s) is newer than the schema version in use by the database (%s) running lily this way will cause models data persistence failures: %w", LatestSchemaVersion(), dbVersion, ErrSchemaTooOld)
	}
	if LatestSchemaVersion().Before(dbVersion) {
		// the latest schema version supported by lily (LatestSchemaVersion()) is older than the schema version in use by the database (`dbVersion`)
		// running lily this way will cause undefined behaviour.
		return model.Version{}, xerrors.Errorf("the latest schema version supported by lily (%s) is older than the schema version in use by the database (%s) running lily this way will cause undefined behaviour: %w", LatestSchemaVersion(), dbVersion, ErrSchemaTooNew)
	}
	return dbVersion, nil
}

// LatestSchemaVersion returns the most recent version of the model schema. It is based on the highest migration version
// in the highest major schema version
func LatestSchemaVersion() model.Version {
	return latestSchemaVersionForMajor(schemas.LatestMajor)
}

// latestSchemaVersionForMajor returns the most recent version of the model schema for a given patch version. It is
// based on the highest migration version
func latestSchemaVersionForMajor(major int) model.Version {
	switch major {
	case 1:
		return v1.Version()
	default:
		return model.Version{} //, xerrors.Errorf("unsupported major version: %d", version.Major)
	}
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

	dbVersion, initialized, err := getDatabaseSchemaVersion(ctx, db, d.SchemaConfig())
	if err != nil {
		return xerrors.Errorf("get schema versions: %w", err)
	}
	log.Infof("current database schema is version %s", dbVersion)

	// Check that we are not trying to migrate to a different major version of an already installed schema
	if initialized && target.Major != dbVersion.Major {
		return xerrors.Errorf("cannot migrate to a different major schema version. database version=%s, target version=%s", dbVersion, target)
	}

	latestVersion := latestSchemaVersionForMajor(target.Major)
	if latestVersion.Patch < target.Patch {
		return xerrors.Errorf("no migrations found for version %s", target)
	}

	if dbVersion == target {
		return xerrors.Errorf("database schema is already at version %d", dbVersion)
	}

	coll, err := collectionForVersion(target, d.SchemaConfig())
	if err != nil {
		return xerrors.Errorf("no schema definition corresponds to version %s: %w", target, err)
	}

	if err := checkMigrationSequence(ctx, coll, dbVersion.Patch, target.Patch); err != nil {
		return xerrors.Errorf("check migration sequence: %w", err)
	}

	// Acquire an exclusive lock on the schema so we know no other instances are running
	if err := SchemaLock.LockExclusive(ctx, db); err != nil {
		return xerrors.Errorf("acquiring schema lock: %w", err)
	}

	if err := initDatabaseSchema(ctx, db, d.SchemaConfig()); err != nil {
		return xerrors.Errorf("initializing schema version tables: %w", err)
	}

	// Check if we need to create the base schema
	if !initialized {
		log.Infof("creating base schema for major version %d", target.Major)

		base, err := baseForVersion(target, d.SchemaConfig())
		if err != nil {
			return xerrors.Errorf("no base schema defined for version %s: %w", target, err)
		}

		if _, err := db.Exec(base); err != nil {
			return xerrors.Errorf("creating base schema: %w", err)
		}

		dbVersion.Major = target.Major
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
			_, newDBPatch, err := coll.Run(db, "down")
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
	_, newDBPatch, err := coll.Run(db, "up", strconv.Itoa(target.Patch))
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

	if from == to {
		return nil
	}

	if from > to {
		to, from = from, to
	}

	for i := from; i <= to; i++ {
		// Migration 0 is always a no-op since it's the base schema
		if i == 0 {
			continue
		}

		if !versions[int64(i)] {
			return xerrors.Errorf("missing migration for schema version %d", i)
		}
	}

	return nil
}

func collectionForVersion(version model.Version, cfg schemas.Config) (*migrations.Collection, error) {
	switch version.Major {
	case 1:
		return v1.GetPatches(cfg)
	default:
		return nil, xerrors.Errorf("unsupported major version: %d", version.Major)
	}
}

func baseForVersion(version model.Version, cfg schemas.Config) (string, error) {
	switch version.Major {
	case 1:
		return v1.GetBase(cfg)
	default:
		return "", xerrors.Errorf("unsupported major version: %d", version.Major)
	}
}
