package storage

import (
	"context"

	"github.com/filecoin-project/lily/model"
)

var _ model.Storage = (*NullStorage)(nil)

// A NullStorage ignores any requests to persist a model
type NullStorage struct {
}

//revive:disable
func (*NullStorage) PersistBatch(ctx context.Context, p ...model.Persistable) error {
	return nil
}

func (*NullStorage) PersistModel(ctx context.Context, m interface{}) error {
	return nil
}
