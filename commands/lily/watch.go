package lily

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/commands"
	"github.com/filecoin-project/sentinel-visor/lens/lily"
)

type watchOps struct {
	confidence int
	tasks      string
	window     time.Duration
}

var watchFlags watchOps

var LilyWatchCmd = &cli.Command{
	Name:   "watch",
	Usage:  "Watch the filecoin blockchain",
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
			Name:                "lily",
			Tasks:               strings.Split(watchFlags.tasks, ","),
			Window:              watchFlags.window,
			Confidence:          watchFlags.confidence,
			RestartDelay:        0,
			RestartOnCompletion: false,
			RestartOnFailure:    false,
			Database: &lily.LilyDatabaseConfig{
				URL:                  commands.VisorCmdFlags.DB,
				Name:                 commands.VisorCmdFlags.Name,
				PoolSize:             commands.VisorCmdFlags.DBPoolSize,
				AllowUpsert:          commands.VisorCmdFlags.DBAllowUpsert,
				AllowSchemaMigration: commands.VisorCmdFlags.DBAllowMigrations,
			},
		}

		watchID, err := lilyAPI.LilyWatch(ctx, cfg)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "Created Watch Job: %d", watchID); err != nil {
			return err
		}
		return nil
	},
}
