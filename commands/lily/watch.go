package lily

import (
	"time"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/lens/lily"
)

var LilyWatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "Start, Stop, and List watches",
	Subcommands: []*cli.Command{
		LilyWatchStartCmd,
	},
}

var LilyWatchStartCmd = &cli.Command{
	Name:  "start",
	Usage: "start a watch against the chain",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name: "confidence",
		},
	},
	Action: func(cctx *cli.Context) error {
		apic, closer, err := GetSentinelNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lotuscli.ReqContext(cctx)

		// TODO: add a config to Lily and inject it to set these configs
		cfg := &lily.LilyWatchConfig{
			Name:       "lily",
			Tasks:      chain.AllTheTasks,
			Window:     builtin.EpochDurationSeconds * time.Second,
			Confidence: 0,
			Database: &lily.LilyDatabaseConfig{
				URL:                  "postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
				Name:                 "lily-database",
				PoolSize:             75,
				AllowUpsert:          false,
				AllowSchemaMigration: false,
			},
		}

		if err := apic.LilyWatchStart(ctx, cfg); err != nil {
			return err
		}
		// wait for user to cancel
		// <-ctx.Done()
		return nil
	},
}
