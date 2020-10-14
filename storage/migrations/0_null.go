package migrations

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	up := batch(`SELECT 1;`)
	down := batch(`SELECT 1;`)
	migrations.MustRegisterTx(up, down)
}
