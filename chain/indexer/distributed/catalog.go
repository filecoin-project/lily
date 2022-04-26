package distributed

import (
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/distributed")

// NewCatalog returns a Catalog configured with the values specified in config.QueueConfig. Error is non-nill if
// config.QueueConfig contains a duplicate queue name.
func NewCatalog(cfg config.QueueConfig) (*Catalog, error) {
	c := &Catalog{
		queues: map[string]config.AsynqRedisConfig{},
	}

	for name, nc := range cfg.Asynq {
		if _, exists := c.queues[name]; exists {
			return nil, xerrors.Errorf("duplicate queue name: %q", name)
		}
		log.Debugw("registering queue", "name", name, "type", "redis")

		c.queues[name] = config.AsynqRedisConfig{
			Network:  nc.Network,
			Addr:     nc.Addr,
			Username: nc.Username,
			Password: nc.Password,
			DB:       nc.DB,
			PoolSize: nc.PoolSize,
		}
	}
	return c, nil
}

// Catalog contains a map of queue names to their configurations. Catalog is used to configure the distributed indexer.
type Catalog struct {
	queues map[string]config.AsynqRedisConfig
}

// AsynqConfig returns a config.AsynqRedisConfig by `name`. And error is returned if name is empty or if a
// config.AsynqRedisConfig doesn't exist for `name`.
func (c *Catalog) AsynqConfig(name string) (config.AsynqRedisConfig, error) {
	if name == "" {
		return config.AsynqRedisConfig{}, xerrors.Errorf("queue config name required")
	}

	n, exists := c.queues[name]
	if !exists {
		return config.AsynqRedisConfig{}, xerrors.Errorf("unknown queue: %q", name)
	}
	return n, nil
}
