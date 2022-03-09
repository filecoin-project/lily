package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens/lily"

	"github.com/filecoin-project/lily/chain"
)

type walkOps struct {
	from     int64
	to       int64
	tasks    string
	window   time.Duration
	storage  string
	apiAddr  string
	apiToken string
	name     string
}

var walkFlags walkOps

var WalkCmd = &cli.Command{
	Name:  "walk",
	Usage: "Start a daemon job to walk a range of the filecoin blockchain.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
			EnvVars:     []string{"LILY_TASKS"},
			Value:       strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			Destination: &walkFlags.tasks,
		},
		&cli.DurationFlag{
			Name:        "window",
			Usage:       "Duration after which any indexing work not completed will be marked incomplete",
			EnvVars:     []string{"LILY_WINDOW"},
			Value:       builtin.EpochDurationSeconds * time.Second * 10, // walks don't need to complete within a single epoch
			Destination: &walkFlags.window,
		},
		&cli.Int64Flag{
			Name:        "from",
			Usage:       "Limit actor and message processing to tipsets at or above `HEIGHT`",
			EnvVars:     []string{"LILY_FROM"},
			Destination: &walkFlags.from,
			Required:    true,
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			EnvVars:     []string{"LILY_TO"},
			Destination: &walkFlags.to,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			EnvVars:     []string{"LILY_STORAGE"},
			Value:       "",
			Destination: &walkFlags.storage,
		},
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &walkFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &walkFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			EnvVars:     []string{"LILY_JOB_NAME"},
			Value:       "",
			Destination: &walkFlags.name,
		},
	},
	Before: func(cctx *cli.Context) error {
		from, to := walkFlags.from, walkFlags.to
		if to < from {
			return xerrors.Errorf("value of --to (%d) should be >= --from (%d)", to, from)
		}

		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		walkName := fmt.Sprintf("walk_%d", time.Now().Unix())
		if walkFlags.name != "" {
			walkName = walkFlags.name
		}

		cfg := &lily.LilyWalkConfig{
			Name:                walkName,
			Tasks:               strings.Split(walkFlags.tasks, ","),
			Window:              walkFlags.window,
			From:                walkFlags.from,
			To:                  walkFlags.to,
			RestartDelay:        0,
			RestartOnCompletion: false,
			RestartOnFailure:    false,
			Storage:             walkFlags.storage,
		}

		api, closer, err := GetAPI(ctx, walkFlags.apiAddr, walkFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyWalk(ctx, cfg)
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}
		return nil
	},
}
