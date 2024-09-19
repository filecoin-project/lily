package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hibiken/asynq"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/indexer"

	"github.com/filecoin-project/lotus/node/config"
)

var log = logging.Logger("lily/config")

// Conf defines the daemon config. It should be compatible with Lotus config.
type Conf struct {
	config.Common
	Chainstore config.Chainstore
	Storage    StorageConf
	Queue      QueueConfig
}

type StorageConf struct {
	Postgresql map[string]PgStorageConf
	File       map[string]FileStorageConf
}

type PgStorageConf struct {
	URLEnv          string // name of an environment variable that contains the database URL
	URL             string // URL used to connect to postgresql if URLEnv is not set
	ApplicationName string
	SchemaName      string
	PoolSize        int
	AllowUpsert     bool
}

type FileStorageConf struct {
	Format      string
	Path        string
	OmitHeader  bool   // when true, don't write column headers to new output files
	FilePattern string // pattern to use for filenames written in the path specified
}

type QueueConfig struct {
	Workers   map[string]AsynqWorkerConfig
	Notifiers map[string]RedisConfig
}

type AsynqWorkerConfig struct {
	RedisConfig  RedisConfig
	WorkerConfig WorkerConfig
}

type RedisConfig struct {
	// Network type to use, either tcp or unix.
	// Default is tcp.
	Network string

	// Redis server address in "host:port" format.
	Addr    string
	AddrEnv string

	// Username to authenticate the current connection when Redis ACLs are used.
	// See: https://redis.io/commands/auth.
	Username    string
	UsernameEnv string

	// Password to authenticate the current connection.
	// See: https://redis.io/commands/auth.
	Password    string
	PasswordEnv string

	// Redis DB to select after connecting to a server.
	// See: https://redis.io/commands/select.
	DB int

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
}

type WorkerConfig struct {
	// Maximum number of concurrent processing of tasks.
	//
	// If set to a zero or negative value, NewServer will overwrite the value
	// to the number of CPUs usable by the current process.
	Concurrency int

	// LogLevel specifies the minimum log level to enable.
	//
	// If unset, InfoLevel is used by default.
	LoggerLevel string

	// Priority is treated as follows to avoid starving low priority queues.
	//
	// Example:
	//
	//	 WatchQueuePriority: 	5
	//	 FillQueuePriority:		3
	//	 IndexQueuePriority:	1
	//	 WalkQueuePriority: 	1
	//
	// With the above config and given that all queues are not empty, the tasks
	// in "watch", "fill", "index", "walk" should be processed 50%, 30%, 10%, 10% of
	// the time respectively.
	WatchQueuePriority int
	FillQueuePriority  int
	IndexQueuePriority int
	WalkQueuePriority  int

	// StrictPriority indicates whether the queue priority should be treated strictly.
	//
	// If set to true, tasks in the queue with the highest priority is processed first.
	// The tasks in lower priority queues are processed only when those queues with
	// higher priorities are empty.
	StrictPriority bool

	// ShutdownTimeout specifies the duration to wait to let workers finish their tasks
	// before forcing them to abort when stopping the server.
	//
	// If unset or zero, default timeout of 8 seconds is used.
	ShutdownTimeout time.Duration
}

func (q WorkerConfig) Queues() map[string]int {
	return map[string]int{
		indexer.Watch.String(): q.WatchQueuePriority,
		indexer.Fill.String():  q.FillQueuePriority,
		indexer.Index.String(): q.IndexQueuePriority,
		indexer.Walk.String():  q.WalkQueuePriority,
	}
}

func (q WorkerConfig) LogLevel() asynq.LogLevel {
	switch strings.ToLower(q.LoggerLevel) {
	case "debug":
		return asynq.DebugLevel
	case "info":
		return asynq.InfoLevel
	case "warn":
		return asynq.WarnLevel
	case "error":
		return asynq.ErrorLevel
	case "fatal":
		return asynq.FatalLevel
	default:
		log.Warnf("invalid log level given (%s) defaulting to level 'INFO'", q.LoggerLevel)
		return asynq.InfoLevel
	}
}

func DefaultConf() *Conf {
	return &Conf{
		Common: config.Common{
			API: config.API{
				ListenAddress: "/ip4/127.0.0.1/tcp/1234/http",
				Timeout:       config.Duration(30 * time.Second),
			},
		},
	}
}

// SampleConf is the example configuration that is written when lily is first started. All entries will be commented out.
func SampleConf() *Conf {
	def := DefaultConf()
	cfg := *def
	cfg.Storage = StorageConf{
		Postgresql: map[string]PgStorageConf{
			"Database1": {
				URLEnv:          "LILY_DB", // LILY_DB is a historical accident, but we keep it as the default for compatibility
				URL:             "postgres://postgres:password@localhost:5432/postgres",
				PoolSize:        20,
				ApplicationName: "visor",
				AllowUpsert:     false,
				SchemaName:      "public",
			},
			// this second database is only here to give an example to the user
			"Database2": {
				URL:             "postgres://postgres:password@localhost:5432/postgres",
				PoolSize:        10,
				ApplicationName: "visor",
				AllowUpsert:     false,
				SchemaName:      "public",
			},
		},

		File: map[string]FileStorageConf{
			"CSV": {
				Format:      "CSV",
				Path:        "/tmp",
				OmitHeader:  false,
				FilePattern: "{table}.csv",
			},
		},
	}
	cfg.Queue = QueueConfig{
		Workers: map[string]AsynqWorkerConfig{
			"Worker1": {
				RedisConfig: RedisConfig{
					Network:     "tcp",
					Addr:        "127.0.0.1:6379",
					Username:    "",
					Password:    "",
					PasswordEnv: "LILY_ASYNQ_REDIS_PASSWORD",
					DB:          0,
					PoolSize:    0,
				},
				WorkerConfig: WorkerConfig{
					Concurrency:        1,
					LoggerLevel:        "debug",
					WatchQueuePriority: 5,
					FillQueuePriority:  3,
					IndexQueuePriority: 1,
					WalkQueuePriority:  1,
					StrictPriority:     false,
					ShutdownTimeout:    time.Second * 30,
				},
			},
		},
		Notifiers: map[string]RedisConfig{
			"Notifier1": {
				Network:     "tcp",
				Addr:        "127.0.0.1:6379",
				Username:    "",
				Password:    "",
				PasswordEnv: "LILY_ASYNQ_REDIS_PASSWORD",
				DB:          0,
				PoolSize:    0,
			},
		},
	}

	return &cfg
}

func EnsureExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	c, err := os.Create(path)
	if err != nil {
		return err
	}
	comm, err := config.ConfigUpdate(SampleConf(), nil, config.Commented(false))
	if err != nil {
		return fmt.Errorf("comment: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		_ = c.Close() // ignore error since we are recovering from a write error anyway
		return fmt.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	return nil
}

// FromFile loads config from a specified file. If file does not exist or is empty defaults are assumed.
func FromFile(path string) (*Conf, error) {
	log.Infof("reading config from %s", path)
	file, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
		log.Warnf("config does not exist at %s, falling back to defaults", path)
		return DefaultConf(), nil
	case err != nil:
		return nil, err
	}

	defer file.Close() //nolint:errcheck // The file is RO
	return FromReader(file, DefaultConf())
}

// FromReader loads config from a reader instance.
func FromReader(reader io.Reader, def *Conf) (*Conf, error) {
	cfg := *def
	if _, err := toml.NewDecoder(reader).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
