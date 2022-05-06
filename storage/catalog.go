package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/model"
)

type Connector interface {
	Connect(context.Context) error
	IsConnected(context.Context) bool
	Close(context.Context) error
}

func NewCatalog(cfg config.StorageConf) (*Catalog, error) {
	c := &Catalog{
		storages: make(map[string]model.Storage),
	}

	for name, sc := range cfg.Postgresql {
		if _, exists := c.storages[name]; exists {
			return nil, fmt.Errorf("duplicate storage name: %q", name)
		}
		log.Debugw("registering storage", "name", name, "type", "postgresql")

		// Find the url of the database, which is either indirectly specified using URLEnv or explicit via URL
		var dburl string
		if sc.URLEnv != "" {
			dburl = os.Getenv(sc.URLEnv)
		} else {
			dburl = sc.URL
		}

		db, err := NewDatabase(context.TODO(), dburl, sc.PoolSize, sc.ApplicationName, sc.SchemaName, sc.AllowUpsert)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgresql storage %q: %w", name, err)
		}

		c.storages[name] = db
	}

	for name, sc := range cfg.File {
		if _, exists := c.storages[name]; exists {
			return nil, fmt.Errorf("duplicate storage name: %q", name)
		}

		switch sc.Format {
		case "CSV":
			log.Debugw("registering storage", "name", name, "type", "csv")

			opts := DefaultCSVStorageOptions()
			opts.OmitHeader = sc.OmitHeader
			opts.FilePattern = sc.FilePattern

			db, err := NewCSVStorageLatest(sc.Path, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to create postgresql storage %q: %w", name, err)
			}
			c.storages[name] = db

		default:
			return nil, fmt.Errorf("unsupported format %q for storage %q", sc.Format, name)
		}

	}

	return c, nil
}

// A Catalog holds a list of pre-configured storage systems and can open them when requested.
type Catalog struct {
	storages map[string]model.Storage
}

// Connect returns a storage that is ready for use. If name is empty, a null storage will be returned
func (c *Catalog) Connect(ctx context.Context, name string, md Metadata) (model.Storage, error) {
	if name == "" {
		return &NullStorage{}, nil
	}

	s, exists := c.storages[name]
	if !exists {
		return nil, fmt.Errorf("unknown storage: %q", name)
	}

	// Does this storage support metadata?
	ms, ok := s.(StorageWithMetadata)
	if ok {
		s = ms.WithMetadata(md)
	}

	// Does this storage need to be connected?
	cs, ok := s.(Connector)
	if ok {
		if !cs.IsConnected(ctx) {
			err := cs.Connect(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

type StorageWithMetadata interface {
	// WithMetadata returns a storage based configured with the supplied metadata
	WithMetadata(Metadata) model.Storage
}

// Metadata is additional information that a storage may use to annotate the data it writes
type Metadata struct {
	JobName string // name of the job using the storage
}

// ConnectAsDatabase returns a storage that is ready to use for reading and writing: `name` must corresponds to
// a Database storage.
func (c *Catalog) ConnectAsDatabase(ctx context.Context, name string, md Metadata) (*Database, error) {
	strg, err := c.Connect(ctx, name, md)
	if err != nil {
		return nil, err
	}

	db, ok := strg.(*Database)
	if !ok {
		return nil, fmt.Errorf("storage type (%T) is unsupported", strg)
	}
	return db, nil
}
