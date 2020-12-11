package storage

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

var _ model.Storage = (*NullStorage)(nil)

// A NullStorage ignores any requests to persist a model
type NullStorage struct {
}

func (*NullStorage) PersistBatch(ctx context.Context, p ...model.Persistable) error {
	log.Debugw("Not persisting data")
	return nil
}

func (*NullStorage) PersistModel(ctx context.Context, m interface{}) error {
	log.Debugw("Not persisting data")
	return nil
}
