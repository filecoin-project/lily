package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

var WalkCmd = &cli.Command{
	Name:  "walk",
	Usage: "Start a daemon job to walk a range of the filecoin blockchain.",
	Flags: []cli.Flag{
		RangeFromFlag,
		RangeToFlag,
	},
	Subcommands: []*cli.Command{
		WalkNotifyCmd,
	},
	Before: func(cctx *cli.Context) error {
		return rangeFlags.validate()
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		cfg := &lily.LilyWalkConfig{
			JobConfig: RunFlags.ParseJobConfig(),
			From:      rangeFlags.from,
			To:        rangeFlags.to,
		}

		res, err := api.LilyWalk(ctx, cfg)
		if err != nil {
			return err
		}

		if err := commands.PrintNewJob(os.Stdout, res); err != nil {
			return err
		}

		return nil
	},
}

var WalkNotifyCmd = &cli.Command{
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

		cfg := &lily.LilyWalkNotifyConfig{
			WalkConfig: lily.LilyWalkConfig{
				JobConfig: RunFlags.ParseJobConfig(),
				From:      rangeFlags.from,
				To:        rangeFlags.to,
			},
			Queue: notifyFlags.queue,
		}

		res, err := api.LilyWalkNotify(ctx, cfg)
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)

	},
}
