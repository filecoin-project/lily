package job

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"

	lotuscli "github.com/filecoin-project/lotus/cli"
)

var tipsetWorkerFlags struct {
	queue string
}

var TipSetWorkerCmd = &cli.Command{
	Name:  "tipset-worker",
	Usage: "start a tipset-worker that consumes tasks from the provided queuing system and performs indexing",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "queue",
			Usage:       "Name of queue system worker will consume work from.",
			EnvVars:     []string{"LILY_TSWORKER_QUEUE"},
			Value:       "",
			Destination: &tipsetWorkerFlags.queue,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.StartTipSetWorker(ctx, &lily.LilyTipSetWorkerConfig{
			JobConfig: RunFlags.ParseJobConfig("tipset-worker"),
			Queue:     tipsetWorkerFlags.queue,
		})
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)
	},
}
