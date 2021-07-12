package v0

import (
	"github.com/go-pg/migrations/v8"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/schemas"
)

const MajorVersion = 0

func init() {
	schemas.RegisterSchema(MajorVersion)
}

// patches is the collection of patches made to the base schema
var patches = migrations.NewCollection()

func GetPatches(cfg schemas.Config) (*migrations.Collection, error) {
	patches.SetTableName(cfg.SchemaName + ".gopg_migrations")
	return patches, nil
}

func Version() model.Version {
	var latestMigration int64
	ms := patches.Migrations()
	for _, m := range ms {
		if m.Version > latestMigration {
			latestMigration = m.Version
		}
	}

	return model.Version{
		Major: MajorVersion,
		Patch: int(latestMigration),
	}
}

// batch is a syntactic helper for registering a migration
func batch(sqls ...string) func(db migrations.DB) error {
	return func(db migrations.DB) error {
		for _, sql := range sqls {
			if _, err := db.Exec(sql); err != nil {
				return err
			}
		}
		return nil
	}
}

// Base is the initial schema for this major version. Patches are applied on top of this base.
var Base = `
	CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public;
`
