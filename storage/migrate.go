package storage

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/schemas"
	v0 "github.com/filecoin-project/sentinel-visor/schemas/v0"
	v1 "github.com/filecoin-project/sentinel-visor/schemas/v1"
)

// GetSchemaVersions returns the schema version in the database and the latest schema version defined by the available
// migrations.
func (d *Database) GetSchemaVersions(ctx context.Context) (model.Version, model.Version, error) {
	latest := LatestSchemaVersion()

	// If we're already connected then use that connection
	if d.db != nil {
		dbVersion, _, err := getDatabaseSchemaVersion(ctx, d.db, d.schemaName)
		return dbVersion, latest, err
	}

	// Temporarily connect
	db, err := connect(ctx, d.opt)
	if err != nil {
		return model.Version{}, model.Version{}, xerrors.Errorf("connect: %w", err)
	}
	defer db.Close() // nolint: errcheck
	dbVersion, _, err := getDatabaseSchemaVersion(ctx, db, d.schemaName)
	return dbVersion, latest, err
}

// getDatabaseSchemaVersion returns the schema version in use by the database and whether the schema versioning
// tables have been initialized. If no schema version tables can be found then the database is assumed to be
// uninitialized and a zero version and false value will be returned. The returned boolean will only be true
// if the schema versioning tables exist and are populated correctly.
func getDatabaseSchemaVersion(ctx context.Context, db *pg.DB, schemaName string) (model.Version, bool, error) {
	vvExists, err := tableExists(ctx, db, schemaName, "visor_version")
	if err != nil {
		return model.Version{}, false, xerrors.Errorf("checking if visor_version exists:%w", err)
	}

	migExists, err := tableExists(ctx, db, schemaName, "gopg_migrations")
	if err != nil {
		return model.Version{}, false, xerrors.Errorf("checking if gopg_migrations exists:%w", err)
	}

	if !migExists && !vvExists {
		// Uninitialized database
		return model.Version{}, false, nil
	}

	// Ensure the visor_version table exists
	vvTableName := schemaName + ".visor_version"
	var major int
	_, err = db.QueryOne(pg.Scan(&major), `SELECT major FROM ? LIMIT 1`, pg.SafeQuery(vvTableName))
	if err != nil && err != pg.ErrNoRows {
		return model.Version{}, false, err
	}

	coll, err := collectionForVersion(model.Version{
		Major: major,
	})
	if err != nil {
		return model.Version{}, false, err
	}
	coll.SetTableName(schemaName + ".gopg_migrations")

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
func initDatabaseSchema(ctx context.Context, db *pg.DB, schemaName string) error {
	if schemaName != "public" {
		_, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS ?`, pg.SafeQuery(schemaName))
		if err != nil {
			return xerrors.Errorf("ensure schema exists :%w", err)
		}
	}

	// Ensure the visor_version table exists
	vvTableName := schemaName + ".visor_version"
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
	migTableName := schemaName + ".gopg_migrations"
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

func validateDatabaseSchemaVersion(ctx context.Context, db *pg.DB, schemaName string) (model.Version, error) {
	// Check if the version of the schema is compatible
	dbVersion, initialized, err := getDatabaseSchemaVersion(ctx, db, schemaName)
	if err != nil {
		return model.Version{}, xerrors.Errorf("get schema version: %w", err)
	}

	if !initialized {
		return model.Version{}, xerrors.Errorf("schema not installed in database")
	}

	latestVersion := LatestSchemaVersion()
	switch {
	case latestVersion.Before(dbVersion):
		// porridge too hot
		return model.Version{}, ErrSchemaTooNew
	case dbVersion.Before(model.OldestSupportedSchemaVersion):
		// porridge too cold
		return model.Version{}, ErrSchemaTooOld
	default:
		// just right
		return dbVersion, nil
	}
}

// LatestSchemaVersion returns the most recent version of the model schema. It is based on the highest migration version
// in the highest major schema version
func LatestSchemaVersion() model.Version {
	return latestSchemaVersionForMajor(schemas.LatestMajor)
}

// latestSchemaVersionForMajor returns the most recent version of the model schema for a given patch version. It is
// based on the highest migration version
func latestSchemaVersionForMajor(major int) model.Version {
	version := model.Version{
		Major: major,
	}

	coll, err := collectionForVersion(version)
	if err != nil {
		panic(fmt.Sprintf("inconsistent schema versions: no patches found for major version %d", version.Major))
	}

	version.Patch = getHighestMigration(coll)
	return version
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
	if target.Major == 0 && d.schemaName != "public" {
		return xerrors.Errorf("v0 schema must use the public postgresql schema")
	}

	db, err := connect(ctx, d.opt)
	if err != nil {
		return xerrors.Errorf("connect: %w", err)
	}
	defer db.Close() // nolint: errcheck

	dbVersion, initialized, err := getDatabaseSchemaVersion(ctx, db, d.schemaName)
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

	coll, err := collectionForVersion(target)
	if err != nil {
		return xerrors.Errorf("no schema definition corresponds to version %s: %w", target, err)
	}
	coll.SetTableName(d.schemaName + ".gopg_migrations")

	if err := checkMigrationSequence(ctx, coll, dbVersion.Patch, target.Patch); err != nil {
		return xerrors.Errorf("check migration sequence: %w", err)
	}

	// Acquire an exclusive lock on the schema so we know no other instances are running
	if err := SchemaLock.LockExclusive(ctx, db); err != nil {
		return xerrors.Errorf("acquiring schema lock: %w", err)
	}

	if err := initDatabaseSchema(ctx, db, d.schemaName); err != nil {
		return xerrors.Errorf("initializing schema version tables: %w", err)
	}

	// Check if we need to create the base schema
	if dbVersion.Patch == 0 {
		log.Infof("creating base schema for major version %d", target.Major)

		cfg := schemas.Config{
			SchemaName: d.schemaName,
		}

		base, err := baseForVersion(target, cfg)
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

func collectionForVersion(version model.Version) (*migrations.Collection, error) {
	switch version.Major {
	case 0:
		return v0.Patches, nil
	case 1:
		return v1.Patches, nil
	default:
		return nil, xerrors.Errorf("unsupported major version: %d", version.Major)
	}
}

func baseForVersion(version model.Version, cfg schemas.Config) (string, error) {
	switch version.Major {
	case 0:
		return v0.Base, nil
	case 1:
		tmpl, err := template.New("base").Funcs(schemaTemplateFuncMap).Parse(v1.BaseTemplate)
		if err != nil {
			return "", xerrors.Errorf("parse base template: %w", err)
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, cfg); err != nil {
			return "", xerrors.Errorf("execute base template: %w", err)
		}
		return buf.String(), nil
	default:
		return "", xerrors.Errorf("unsupported major version: %d", version.Major)
	}
}

func isEmpty(val interface{}) bool {
	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		return false
	default:
		return v.IsNil()
	}
}

var schemaTemplateFuncMap = template.FuncMap{
	"default": func(def interface{}, value interface{}) interface{} {
		if isEmpty(value) {
			return def
		}
		return value
	},
}
