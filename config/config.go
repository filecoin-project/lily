package config

import (
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/filecoin-project/lotus/node/config"
	"golang.org/x/xerrors"
)

// Conf defines the daemon config. It should be compatible with Lotus config.
type Conf struct {
	config.Common
	Client     config.Client
	Metrics    config.Metrics
	Chainstore config.Chainstore
	Storage    StorageConf
}

type StorageConf struct {
	Postgresql map[string]PgStorageConf
	File       map[string]FileStorageConf
}

type PgStorageConf struct {
	URLEnv          string // name of an environment variable that contains the database URL
	URL             string // URL used to connect to postgresql if URLEnv is not set
	ApplicationName string
	PoolSize        int
	AllowUpsert     bool
}

type FileStorageConf struct {
	Format string
	Path   string
}

func DefaultConf() *Conf {
	return &Conf{
		Common: config.Common{
			API: config.API{
				ListenAddress: "/ip4/127.0.0.1/tcp/1234/http",
				Timeout:       config.Duration(30 * time.Second),
			},
			Libp2p: config.Libp2p{
				ListenAddresses: []string{
					"/ip4/0.0.0.0/tcp/0",
					"/ip6/::/tcp/0",
				},
				AnnounceAddresses:   []string{},
				NoAnnounceAddresses: []string{},

				ConnMgrLow:   150,
				ConnMgrHigh:  180,
				ConnMgrGrace: config.Duration(20 * time.Second),
			},
			Pubsub: config.Pubsub{
				Bootstrapper: false,
				DirectPeers:  nil,
				RemoteTracer: "/dns4/pubsub-tracer.filecoin.io/tcp/4001/p2p/QmTd6UvR47vUidRNZ1ZKXHrAFhqTJAD27rKL9XYghEKgKX",
			},
		},
		Client: config.Client{
			SimultaneousTransfers: config.DefaultSimultaneousTransfers,
		},
		Storage: StorageConf{
			Postgresql: map[string]PgStorageConf{
				"Database1": {
					URLEnv:          "LOTUS_DB", // LOTUS_DB is a historical accident, but we keep it as the default for compatibility
					URL:             "postgres://postgres:password@localhost:5432/postgres",
					PoolSize:        20,
					ApplicationName: "visor",
					AllowUpsert:     false,
				},
				// this second database is only here to give an example to the user
				"Database2": {
					URL:             "postgres://postgres:password@localhost:5432/postgres",
					PoolSize:        10,
					ApplicationName: "visor",
					AllowUpsert:     false,
				},
			},

			File: map[string]FileStorageConf{
				"CSV": {
					Format: "CSV",
					Path:   "/tmp",
				},
			},
		},
	}
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

	comm, err := config.ConfigComment(DefaultConf())
	if err != nil {
		return xerrors.Errorf("comment: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		_ = c.Close() // ignore error since we are recovering from a write error anyway
		return xerrors.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return xerrors.Errorf("close config: %w", err)
	}
	return nil
}

// FromFile loads config from a specified file. If file does not exist or is empty defaults are assumed.
func FromFile(path string) (*Conf, error) {
	file, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
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
	_, err := toml.DecodeReader(reader, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
