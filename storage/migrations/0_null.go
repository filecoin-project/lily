package migrations

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	up := batch(``)
	down := batch(``)
	migrations.MustRegisterTx(up, down)
}
