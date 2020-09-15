package model

import (
	"context"
	"github.com/go-pg/pg/v10"
)

type Persistable interface {
	Persist(ctx context.Context, db *pg.DB) error
}
