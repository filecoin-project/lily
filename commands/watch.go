package commands

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/lens/lily"
)

type watchOps struct {
	confidence int
	workers    int
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
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		cfg := &lily.LilyWatchConfig{
			JobConfig:  jobConfigFromFlags(cctx, runFlags),
			Confidence: watchFlags.confidence,
			Workers:    watchFlags.workers,
		}

		api, closer, err := GetAPI(ctx, runFlags.apiAddr, runFlags.apiToken)
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
