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
		var redisAddr string
		if sc.RedisConfig.AddrEnv != "" {
			redisAddr = os.Getenv(sc.RedisConfig.AddrEnv)
		} else {
			redisAddr = sc.RedisConfig.Addr
		}
		var redisUser string
		if sc.RedisConfig.UsernameEnv != "" {
			redisUser = os.Getenv(sc.RedisConfig.UsernameEnv)
		} else {
			redisUser = sc.RedisConfig.Username
		}
		var redisPassword string
		if sc.RedisConfig.PasswordEnv != "" {
			redisPassword = os.Getenv(sc.RedisConfig.PasswordEnv)
		} else {
			redisPassword = sc.RedisConfig.Password
		}

		c.servers[name] = &TipSetWorker{
			RedisConfig: asynq.RedisClientOpt{
				Network:  sc.RedisConfig.Network,
				Addr:     redisAddr,
				Username: redisUser,
				Password: redisPassword,
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
		var redisAddr string
		if cc.AddrEnv != "" {
			redisAddr = os.Getenv(cc.AddrEnv)
		} else {
			redisAddr = cc.Addr
		}
		var redisUser string
		if cc.UsernameEnv != "" {
			redisUser = os.Getenv(cc.UsernameEnv)
		} else {
			redisUser = cc.Username
		}
		var redisPassword string
		if cc.PasswordEnv != "" {
			redisPassword = os.Getenv(cc.PasswordEnv)
		} else {
			redisPassword = cc.Password
		}

		c.clients[name] = asynq.NewClient(
			asynq.RedisClientOpt{
				Network:  cc.Network,
				Addr:     redisAddr,
				Username: redisUser,
				Password: redisPassword,
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
