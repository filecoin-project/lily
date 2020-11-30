package model

import (
	"context"

	"github.com/go-pg/pg/v10"
)

type noData struct{}

func (noData) Persist(ctx context.Context, db *pg.DB) error {
	return nil
}

// NoData is a model with no data to persist.
var NoData = noData{}
