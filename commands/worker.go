package commands

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/lens/lily"
)

var WorkerCmd = &cli.Command{
	Name: "worker-start",
	Subcommands: []*cli.Command{
		TipSetWorkerCmd,
	},
}

var tipsetWorkerFlags struct {
	queue       string
	name        string
	storage     string
	concurrency int
}

var TipSetWorkerCmd = &cli.Command{
	Name: "tipset-processor",
	Flags: flagSet(
		clientAPIFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "concurrency",
				Usage:       "Concurrency sets the maximum number of concurrent processing of tasks. If set to a zero or negative value it will be set to the number of CPUs usable by the current process.",
				Value:       1,
				Destination: &tipsetWorkerFlags.concurrency,
			},
			&cli.StringFlag{
				Name:        "storage",
				Usage:       "Name of storage that results will be written to.",
				EnvVars:     []string{"LILY_STORAGE"},
				Value:       "",
				Destination: &tipsetWorkerFlags.storage,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of job for easy identification later.",
				EnvVars:     []string{"LILY_JOB_NAME"},
				Value:       "",
				Destination: &tipsetWorkerFlags.name,
			},
			&cli.StringFlag{
				Name:        "queue",
				Usage:       "Name of queue worker will consume work from.",
				EnvVars:     []string{"LILY_TSWORKER_QUEUE"},
				Value:       "",
				Destination: &tipsetWorkerFlags.queue,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		if tipsetWorkerFlags.name == "" {
			id, err := api.ID(ctx)
			if err != nil {
				return err
			}
			tipsetWorkerFlags.name = id.ShortString()
		}

		cfg := &lily.LilyTipSetWorkerConfig{
			Concurrency:         tipsetWorkerFlags.concurrency,
			Storage:             tipsetWorkerFlags.storage,
			Name:                tipsetWorkerFlags.name,
			RestartOnFailure:    true,
			RestartOnCompletion: false,
			RestartDelay:        0,
			Queue:               tipsetWorkerFlags.queue,
		}

		res, err := api.StartTipSetWorker(ctx, cfg)
		if err != nil {
			return err
		}

		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil
	},
}
