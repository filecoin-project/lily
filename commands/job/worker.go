package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

var tipsetWorkerFlags struct {
	queue       string
	concurrency int
}

var TipSetWorkerCmd = &cli.Command{
	Name: "tipset-worker",
	Flags: commands.FlagSet(
		commands.ClientAPIFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "concurrency",
				Usage:       "Concurrency sets the maximum number of concurrent processing of tasks. If set to a zero or negative value it will be set to the number of CPUs usable by the current process.",
				Value:       1,
				Destination: &tipsetWorkerFlags.concurrency,
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

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.StartTipSetWorker(ctx, &lily.LilyTipSetWorkerConfig{
			JobConfig:   RunFlags.ParseJobConfig(),
			Queue:       tipsetWorkerFlags.queue,
			Concurrency: tipsetWorkerFlags.concurrency,
		})
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)
	},
}
