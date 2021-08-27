package model

import (
	"context"
	"errors"
)

// OldestSupportedSchemaVersion is the oldest version of the schema that lily can convert its models to
// Models can be persisted using any version between this and the latest version. Version 28 is the version
// in which support for multiple schemas was introduced.
var OldestSupportedSchemaVersion = Version{Major: 0, Patch: 28}

// ErrUnsupportedSchemaVersion is returned when a Persistable model cannot be persisted in a particular
// schema version.
var ErrUnsupportedSchemaVersion = errors.New("model does not support requested schema version")

// A Storage can marshal models into a serializable format and persist them.
type Storage interface {
	PersistBatch(ctx context.Context, ps ...Persistable) error
}

// A StorageBatch persists a model to storage as part of a batch such as a transaction.
type StorageBatch interface {
	PersistModel(ctx context.Context, m interface{}) error
}

// A Persistable can persist a full copy of itself or its components as part of a storage batch using a specific
// version of a schema. Persist should call PersistModel on s with a model containing data that should be persisted.
// ErrUnsupportedSchemaVersion should be retuned if the Persistable cannot provide a model compatible with the requested
// schema version. If the model does not exist in the schema version because it has been removed or was added in a later
// version then Persist should be a no-op and return nil.
type Persistable interface {
	Persist(ctx context.Context, s StorageBatch, version Version) error
}

// A PersistableList is a list of Persistables that should be persisted together
type PersistableList []Persistable

// Ensure that a PersistableList can be used as a Persistable
var _ Persistable = (PersistableList)(nil)

func (pl PersistableList) Persist(ctx context.Context, s StorageBatch, version Version) error {
	if len(pl) == 0 {
		return nil
	}
	for _, p := range pl {
		if p == nil {
			continue
		}
		if err := p.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}
