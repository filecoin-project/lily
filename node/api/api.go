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
	LilyWatchCreate(ctx context.Context, cfg *LilyWatchConfig, start bool) (int, error)
	LilyWatchStart(ctx context.Context, ID int) error
	LilyWatchStop(ctx context.Context, ID int) error
	LilyWatchList(ctx context.Context) (LilyListResult, error)
}

type LilyListResult struct {
	Result []LilyWatchStatus
}

type LilyWatchStatus struct {
	ID     int
	Status string
	Config *LilyDatabaseConfig
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
