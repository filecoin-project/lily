package queue

import (
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/queue")

type RedisConfig struct {
	// Network type to use, either tcp or unix.
	// Default is tcp.
	Network string
	// Redis server address in "host:port" format.
	Addr string
	// Username to authenticate the current connection when Redis ACLs are used.
	// See: https://redis.io/commands/auth.
	Username string
	// Password to authenticate the current connection.
	// See: https://redis.io/commands/auth.
	Password string
	// Redis DB to select after connecting to a server.
	// See: https://redis.io/commands/select.
	DB int
	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
}

func NewCatalog(cfg config.QueueConfig) (*Catalog, error) {
	c := &Catalog{
		queues: map[string]*RedisConfig{},
	}

	for name, nc := range cfg.Redis {
		if _, exists := c.queues[name]; exists {
			return nil, xerrors.Errorf("duplicate queue name: %q", name)
		}
		log.Debugw("registering queue", "name", name, "type", "redis")

		c.queues[name] = &RedisConfig{
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
	queues map[string]*RedisConfig
}

func (c *Catalog) Config(name string) (*RedisConfig, error) {
	if name == "" {
		return nil, xerrors.Errorf("queue config name required")
	}

	n, exists := c.queues[name]
	if !exists {
		return nil, xerrors.Errorf("unknown queue: %q", name)
	}
	return n, nil
}
