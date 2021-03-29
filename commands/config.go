package commands

import (
	"os"
	"time"

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
	URL             string
	PoolSize        int
	AllowUpsert     bool
	AllowMigrations bool
}

type FileStorageConf struct {
	Format string
	Path   string
}

func defaultConf() Conf {
	return Conf{
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
					URL:             "postgres://postgres:password@localhost:5432/postgres",
					PoolSize:        20,
					AllowUpsert:     false,
					AllowMigrations: false,
				},
				"Database2": {
					URL:             "postgres://postgres:password@localhost:5432/postgres",
					PoolSize:        10,
					AllowUpsert:     false,
					AllowMigrations: false,
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

func initConfig(path string) error {
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

	comm, err := config.ConfigComment(defaultConf())
	if err != nil {
		return xerrors.Errorf("comment: %w", err)
	}
	_, err = c.Write(comm)
	if err != nil {
		return xerrors.Errorf("write config: %w", err)
	}

	if err := c.Close(); err != nil {
		return xerrors.Errorf("close config: %w", err)
	}
	return nil
}
