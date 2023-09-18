package model

import (
	"context"
)

type noData struct{}

func (noData) Persist(_ context.Context, _ StorageBatch, _ Version) error {
	return nil
}

// NoData is a model with no data to persist.
var NoData = noData{}

var _ Persistable = noData{}
