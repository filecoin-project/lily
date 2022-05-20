package distributed

import (
	"fmt"
	"os"

	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/config"
)

var log = logging.Logger("lily/distributed")

// NewCatalog returns a Catalog configured with the values specified in config.QueueConfig. Error is non-nill if
// config.QueueConfig contains a duplicate queue name.
func NewCatalog(cfg config.QueueConfig) (*Catalog, error) {
	c := &Catalog{
		servers: map[string]*TipSetWorker{},
		clients: map[string]*asynq.Client{},
	}

	for name, sc := range cfg.Workers {
		if _, exists := c.servers[name]; exists {
			return nil, fmt.Errorf("duplicate queue name: %q", name)
		}
		log.Infow("registering worker queue config", "name", name, "type", "redis", "addr", sc.RedisConfig.Addr)

		// Find the password of the queue, which is either indirectly specified using PasswordEnv or explicit via Password.
		// TODO use github.com/kelseyhightower/envconfig
		var queuePassword string
		if sc.RedisConfig.PasswordEnv != "" {
			queuePassword = os.Getenv(sc.RedisConfig.PasswordEnv)
		} else {
			queuePassword = sc.RedisConfig.Password
		}

		c.servers[name] = &TipSetWorker{
			RedisConfig: asynq.RedisClientOpt{
				Network:  sc.RedisConfig.Network,
				Addr:     sc.RedisConfig.Addr,
				Username: sc.RedisConfig.Username,
				Password: queuePassword,
				DB:       sc.RedisConfig.DB,
				PoolSize: sc.RedisConfig.PoolSize,
			},
			ServerConfig: asynq.Config{
				LogLevel:        sc.WorkerConfig.LogLevel(),
				Queues:          sc.WorkerConfig.Queues(),
				ShutdownTimeout: sc.WorkerConfig.ShutdownTimeout,
				Concurrency:     sc.WorkerConfig.Concurrency,
				StrictPriority:  sc.WorkerConfig.StrictPriority,
			},
		}
	}

	for name, cc := range cfg.Notifiers {
		if _, exists := c.servers[name]; exists {
			return nil, fmt.Errorf("duplicate queue name: %q", name)
		}
		log.Infow("registering notifier queue config", "name", name, "type", "redis", "addr", cc.Addr)

		// Find the password of the queue, which is either indirectly specified using PasswordEnv or explicit via Password.
		// TODO use github.com/kelseyhightower/envconfig
		var queuePassword string
		if cc.PasswordEnv != "" {
			queuePassword = os.Getenv(cc.PasswordEnv)
		} else {
			queuePassword = cc.Password
		}

		c.clients[name] = asynq.NewClient(
			asynq.RedisClientOpt{
				Network:  cc.Network,
				Addr:     cc.Addr,
				Username: cc.Username,
				Password: queuePassword,
				DB:       cc.DB,
				PoolSize: cc.PoolSize,
			},
		)
	}
	return c, nil
}

type TipSetWorker struct {
	RedisConfig  asynq.RedisClientOpt
	ServerConfig asynq.Config
}

// Catalog contains a map of workers and clients
// Catalog is used to configure the distributed indexer.
type Catalog struct {
	servers map[string]*TipSetWorker
	clients map[string]*asynq.Client
}

// Worker returns a runnable *asynq.Server by `name`. An error is returned if name is empty or if a
// *asynq.Server doesn't exist for `name`.
func (c *Catalog) Worker(name string) (*TipSetWorker, error) {
	if name == "" {
		return nil, fmt.Errorf("server config name required")
	}

	server, exists := c.servers[name]
	if !exists {
		return nil, fmt.Errorf("unknown server: %q", name)
	}
	return server, nil
}

// Notifier returns a *asynq.Client by `name`. An error is returned if name is empty or if a
// *asynq.Client doesn't exist for `name`.
func (c *Catalog) Notifier(name string) (*asynq.Client, error) {
	if name == "" {
		return nil, fmt.Errorf("client config name required")
	}

	client, exists := c.clients[name]
	if !exists {
		return nil, fmt.Errorf("unknown client: %q", name)
	}
	return client, nil
}
