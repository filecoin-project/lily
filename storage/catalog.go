package storage

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/config"
	"github.com/filecoin-project/sentinel-visor/model"
)

func CatalogConstructor(cfg config.StorageConf) func() *Catalog {
	panic("not implemented yet")
}

// A Catalog holds a list of pre-configured storage systems and can open them when requested.
type Catalog struct {
}

func (c *Catalog) Open(ctx context.Context, name string) (model.Storage, error) {
	panic("not implemented yet")
}
