package migrations

import (
	"github.com/go-pg/migrations/v8"
)

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
