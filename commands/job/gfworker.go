package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

var gapFillWorkerFlags struct {
	queue string
}

var GapFillWorkerCmd = &cli.Command{
	Name:  "gapfill-worker",
	Usage: "start a gapfill-worker that consumes tasks from the provided queuing system and performs gap filling",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "queue",
			Usage:       "Name of queue system worker will consume work from.",
			EnvVars:     []string{"LILY_GAPFILL_WORKER_QUEUE"},
			Value:       "",
			Destination: &gapFillWorkerFlags.queue,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.StartGapFillWorker(ctx, &lily.LilyGapFillWorkerConfig{
			JobConfig: RunFlags.ParseJobConfig("gapfill-worker"),
			Queue:     gapFillWorkerFlags.queue,
		})
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)
	},
}
