package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens/lily"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

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
			Value:       strings.Join([]string{chain.BlocksTask, chain.MessagesTask, chain.ChainEconomicsTask, chain.ActorStatesRawTask}, ","),
			Destination: &walkFlags.tasks,
		},
		&cli.DurationFlag{
			Name:        "window",
			Usage:       "Duration after which any indexing work not completed will be marked incomplete",
			Value:       builtin.EpochDurationSeconds * time.Second * 10, // walks don't need to complete within a single epoch
			Destination: &walkFlags.window,
		},
		&cli.Int64Flag{
			Name:        "from",
			Usage:       "Limit actor and message processing to tipsets at or above `HEIGHT`",
			Destination: &walkFlags.from,
			Required:    true,
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			Destination: &walkFlags.to,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			Value:       "",
			Destination: &walkFlags.storage,
		},
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of visor api in multiaddr format.",
			EnvVars:     []string{"VISOR_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &walkFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for visor api.",
			EnvVars:     []string{"VISOR_API_TOKEN"},
			Value:       "",
			Destination: &walkFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			Value:       "",
			Destination: &walkFlags.name,
		},
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

		watchID, err := api.LilyWalk(ctx, cfg)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "Created walk job %d\n", watchID); err != nil {
			return err
		}
		return nil
	},
}
