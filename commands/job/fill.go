package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

var GapFillCmd = &cli.Command{
	Name:  "fill",
	Usage: "Fill gaps in the database",
	Flags: []cli.Flag{
		RangeFromFlag,
		RangeToFlag,
	},
	Subcommands: []*cli.Command{
		GapFillNotifyCmd,
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

		res, err := api.LilyGapFill(ctx, &lily.LilyGapFillConfig{
			JobConfig: RunFlags.ParseJobConfig(),
			To:        rangeFlags.to,
			From:      rangeFlags.from,
		})
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)
	},
}

var GapFillNotifyCmd = &cli.Command{
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

		cfg := &lily.LilyGapFillNotifyConfig{
			GapFillConfig: lily.LilyGapFillConfig{
				JobConfig: RunFlags.ParseJobConfig(),
				From:      rangeFlags.from,
				To:        rangeFlags.to,
			},
			Queue: notifyFlags.queue,
		}

		res, err := api.LilyGapFillNotify(ctx, cfg)
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)

	},
}
