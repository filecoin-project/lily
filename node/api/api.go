package api

import (
	"context"
	"time"

	"github.com/filecoin-project/lotus/api"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sentinel")

type LilyNode interface {
	api.FullNode
	LilyWatchStart(ctx context.Context, cfg *LilyWatchConfig) error
}

type LilyWatchConfig struct {
	Name       string
	Tasks      []string
	Window     time.Duration
	Confidence int
	Database   *LilyDatabaseConfig
}

type LilyDatabaseConfig struct {
	URL  string
	Name string

	PoolSize int

	AllowUpsert          bool
	AllowSchemaMigration bool
}
