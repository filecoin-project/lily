package model

import (
	"context"
)

// A Storage can marshal models into a serializable format and persist them.
type Storage interface {
	PersistBatch(ctx context.Context, ps ...Persistable) error
}

// A StorageBatch persists a model to storage as part of a batch such as a transaction.
type StorageBatch interface {
	PersistModel(ctx context.Context, m interface{}) error
}

// A Persistable can persist a full copy of itself or its components as part of a storage batch
type Persistable interface {
	Persist(ctx context.Context, s StorageBatch) error
}

// A PersistableList is a list of Persistables that should be persisted together
type PersistableList []Persistable

// Ensure that a PersistableList can be used as a Persistable
var _ Persistable = (PersistableList)(nil)

func (pl PersistableList) Persist(ctx context.Context, s StorageBatch) error {
	if len(pl) == 0 {
		return nil
	}
	for _, p := range pl {
		if p == nil {
			continue
		}
		if err := p.Persist(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
