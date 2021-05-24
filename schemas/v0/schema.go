package v0

import (
	"github.com/filecoin-project/sentinel-visor/schemas"
	"github.com/go-pg/migrations/v8"
)

// Patches is the collection of patches made to the base schema
var Patches = migrations.NewCollection()

func init() {
	schemas.RegisterSchema(0)
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
