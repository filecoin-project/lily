package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/index"
	"github.com/filecoin-project/lily/lens/lily"
)

type watchOps struct {
	confidence int
	tasks      string
	window     time.Duration
	storage    string
	apiAddr    string
	apiToken   string
	name       string
	workers    int
	queue      string
}

var watchFlags watchOps

var WatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "Start a daemon job to watch the head of the filecoin blockchain.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "confidence",
			Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
			EnvVars:     []string{"LILY_CONFIDENCE"},
			Value:       2,
			Destination: &watchFlags.confidence,
		},
		&cli.IntFlag{
			Name:        "workers",
			Usage:       "Sets the number of tipsets that may be simultaneous indexed while watching",
			EnvVars:     []string{"LILY_WATCH_WORKERS"},
			Value:       4,
			Destination: &watchFlags.workers,
		},
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
			EnvVars:     []string{"LILY_TASKS"},
			Value:       strings.Join([]string{index.BlocksTask, index.MessagesTask, index.ReceiptTask, index.ChainEconomicsTask, index.ActorStatesRawTask}, ","),
			Destination: &watchFlags.tasks,
		},
		&cli.DurationFlag{
			Name:        "window",
			Usage:       "Duration after which any indexing work not completed will be marked incomplete",
			EnvVars:     []string{"LILY_WINDOW"},
			Value:       builtin.EpochDurationSeconds * time.Second,
			Destination: &watchFlags.window,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			EnvVars:     []string{"LILY_STORAGE"},
			Value:       "",
			Destination: &watchFlags.storage,
		},
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &watchFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &watchFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			EnvVars:     []string{"LILY_JOB_NAME"},
			Value:       "",
			Destination: &watchFlags.name,
		},
		&cli.StringFlag{
			Name:        "queue",
			Usage:       "Name of queue that watcher will write tipsets to.",
			EnvVars:     []string{"LILY_WATCH_QUEUE"},
			Value:       "",
			Destination: &watchFlags.queue,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		watchName := fmt.Sprintf("watch_%d", time.Now().Unix())
		if watchFlags.name != "" {
			watchName = watchFlags.name
		}

		tasks := strings.Split(watchFlags.tasks, ",")
		if watchFlags.tasks == "*" {
			tasks = index.AllTasks
		}

		cfg := &lily.LilyWatchConfig{
			Name:                watchName,
			Tasks:               tasks,
			Window:              watchFlags.window,
			Confidence:          watchFlags.confidence,
			RestartDelay:        0,
			RestartOnCompletion: false,
			RestartOnFailure:    true,
			Storage:             watchFlags.storage,
			Workers:             watchFlags.workers,
			Queue:               watchFlags.queue,
		}

		api, closer, err := GetAPI(ctx, watchFlags.apiAddr, watchFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyWatch(ctx, cfg)
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil
	},
}
