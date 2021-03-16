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

type walkOps struct {
	confidence int
	from       int64
	to         int64
	tasks      string
	window     time.Duration
}

var walkFlags walkOps

var LilyWalkCmd = &cli.Command{
	Name:   "walk",
	Usage:  "walk the filecoin blockchain",
	Before: initialize,
	After:  destroy,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "indexhead-confidence",
			Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
			Value:       2,
			EnvVars:     []string{"SENTINEL_LILY_WALK_CONFIDENCE"},
			Destination: &walkFlags.confidence,
		},
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
			Value:       strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			EnvVars:     []string{"SENTINEL_LILY_WALK_TASKS"},
			Destination: &walkFlags.tasks,
		},
		&cli.DurationFlag{
			Name:        "window",
			Usage:       "Duration after which any indexing work not completed will be marked incomplete",
			Value:       builtin.EpochDurationSeconds * time.Second,
			EnvVars:     []string{"SENTINEL_LILY_WALK_WINDOW"},
			Destination: &walkFlags.window,
		},
		&cli.Int64Flag{
			Name:        "from",
			Usage:       "Limit actor and message processing to tipsets at or above `HEIGHT`",
			EnvVars:     []string{"SENTINEL_LILLY_WALK_HEIGHT_FROM"},
			Destination: &walkFlags.from,
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			Value:       estimateCurrentEpoch(),
			DefaultText: "MaxInt64",
			EnvVars:     []string{"SENTINEL_LILLY_WALK_HEIGHT_TO"},
			Destination: &walkFlags.to,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		cfg := &lily.LilyWalkConfig{
			Name:                "lily",
			Tasks:               strings.Split(walkFlags.tasks, ","),
			Window:              walkFlags.window,
			Confidence:          walkFlags.confidence,
			From:                walkFlags.from,
			To:                  walkFlags.to,
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

		watchID, err := lilyAPI.LilyWalk(ctx, cfg)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "Created Watch Job: %d", watchID); err != nil {
			return err
		}
		return nil
	},
}

var mainnetGenesis = time.Date(2020, 8, 24, 22, 0, 0, 0, time.UTC)

func estimateCurrentEpoch() int64 {
	return int64(time.Since(mainnetGenesis) / (builtin.EpochDurationSeconds))
}
