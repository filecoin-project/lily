package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/schedule"
)

type watchOps struct {
	confidence int
	workers    int
	bufferSize int
}

var watchFlags watchOps

var WatchConfidenceFlag = &cli.IntFlag{
	Name:        "confidence",
	Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database",
	EnvVars:     []string{"LILY_CONFIDENCE"},
	Value:       2,
	Destination: &watchFlags.confidence,
}
var WatchWorkersFlag = &cli.IntFlag{
	Name:        "workers",
	Usage:       "Sets the number of tipsets that may be simultaneous indexed while watching",
	EnvVars:     []string{"LILY_WATCH_WORKERS"},
	Value:       2,
	Destination: &watchFlags.workers,
}
var WatchBufferSizeFlag = &cli.IntFlag{
	Name:        "buffer-size",
	Usage:       "Set the number of tipsets the watcher will buffer while waiting for a worker to accept the work",
	EnvVars:     []string{"LILY_WATCH_BUFFER"},
	Value:       5,
	Destination: &watchFlags.bufferSize,
}

var WatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "Start a daemon job to watch the head of the filecoin blockchain.",
	Flags: []cli.Flag{
		WatchConfidenceFlag,
		WatchWorkersFlag,
		WatchBufferSizeFlag,
	},
	Subcommands: []*cli.Command{
		WatchNotifyCmd,
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		var res *schedule.JobSubmitResult
		cfg := &lily.LilyWatchConfig{
			JobConfig:  RunFlags.ParseJobConfig(),
			BufferSize: watchFlags.bufferSize,
			Confidence: watchFlags.confidence,
			Workers:    watchFlags.workers,
		}

		res, err = api.LilyWatch(ctx, cfg)
		if err != nil {
			return err
		}

		if err := commands.PrintNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil
	},
}

var WatchNotifyCmd = &cli.Command{
	Name: "notify",
	Flags: []cli.Flag{
		NotifyQueueFlag,
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		cfg := &lily.LilyWatchNotifyConfig{
			JobConfig: RunFlags.ParseJobConfig(),

			Confidence: watchFlags.confidence,
			BufferSize: watchFlags.bufferSize,

			Queue: notifyFlags.queue,
		}

		res, err := api.LilyWatchNotify(ctx, cfg)
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)

	},
}
