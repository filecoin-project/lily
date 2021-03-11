package lily

import (
	"context"
	"time"

	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/sentinel-visor/lens"
)

type LilyAPI interface {
	lens.API

	AuthVerify(ctx context.Context, token string) ([]auth.Permission, error)

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
