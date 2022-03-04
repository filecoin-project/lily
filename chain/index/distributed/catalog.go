package distributed

import (
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/distributed")

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

type Catalog struct {
	queues map[string]config.AsynqRedisConfig
}

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
