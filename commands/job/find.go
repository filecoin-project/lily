package job

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

var GapFindCmd = &cli.Command{
	Name:  "find",
	Usage: "find gaps in the database",
	Flags: []cli.Flag{
		RangeFromFlag,
		RangeToFlag,
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

		res, err := api.LilyGapFind(ctx, &lily.LilyGapFindConfig{
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
