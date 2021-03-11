package lily

import (
	"strings"
	"time"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/commands"
	"github.com/filecoin-project/sentinel-visor/lens/lily"
)

var LilyWatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "Start, Stop, and List watches",
	Subcommands: []*cli.Command{
		LilyWatchStartCmd,
	},
}

type watchOps struct {
	confidence int
	tasks      string
	window     time.Duration
}

var watchFlags watchOps

var LilyWatchStartCmd = &cli.Command{
	Name:   "start",
	Usage:  "start a watch against the chain",
	Before: initialize,
	After:  destroy,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "indexhead-confidence",
			Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
			Value:       2,
			EnvVars:     []string{"SENTINEL_LILY_WATCH_CONFIDENCE"},
			Destination: &watchFlags.confidence,
		},
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
			Value:       strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			EnvVars:     []string{"SENTINEL_LILY_WATCH_TASKS"},
			Destination: &watchFlags.tasks,
		},
		&cli.DurationFlag{
			Name:        "window",
			Usage:       "Duration after which any indexing work not completed will be marked incomplete",
			Value:       builtin.EpochDurationSeconds * time.Second,
			EnvVars:     []string{"SENTINEL_LILY_WATCH_WINDOW"},
			Destination: &watchFlags.window,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		cfg := &lily.LilyWatchConfig{
			Name:       "lily",
			Tasks:      strings.Split(watchFlags.tasks, ","),
			Window:     watchFlags.window,
			Confidence: watchFlags.confidence,
			Database: &lily.LilyDatabaseConfig{
				URL:                  commands.VisorCmdFlags.DB,
				Name:                 commands.VisorCmdFlags.Name,
				PoolSize:             commands.VisorCmdFlags.DBPoolSize,
				AllowUpsert:          commands.VisorCmdFlags.DBAllowUpsert,
				AllowSchemaMigration: commands.VisorCmdFlags.DBAllowMigrations,
			},
		}

		if err := lilyAPI.LilyWatchStart(ctx, cfg); err != nil {
			return err
		}
		// wait for user to cancel
		// <-ctx.Done()
		return nil
	},
}
