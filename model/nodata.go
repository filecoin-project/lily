package model

import (
	"context"
)

type noData struct{}

func (noData) Persist(ctx context.Context, s StorageBatch, version Version) error {
	return nil
}

// NoData is a model with no data to persist.
var NoData = noData{}

var _ Persistable = noData{}
