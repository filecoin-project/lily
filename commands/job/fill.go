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
	Usage: "fill gaps in the database for a given range and set of tasks.",
	Description: `
The fill job queries the visor_gap_reports table for gaps to fill and indexes the data reported to have gaps.
A gap in the visor_gap_reports table is any row with status 'GAP'.
fill will index gaps based on the list of tasks (--tasks) provided over the specified range (--from --to).
Each epoch and its corresponding list of tasks found in the visor_gap_reports table will be indexed independently.
When the gap is successfully filled its corresponding entry in the visor_gap_reports table will be updated with status 'FILLED'.

As an example, the below command:
  $ lily job run --tasks=block_headers,message fill --from=10 --to=20
fills gaps for block_headers and messages tasks from epoch 10 to 20 (inclusive)

Constraints:
- the fill job must be executed AFTER a find job. These jobs must NOT be executed simultaneously.
`,
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
			JobConfig: RunFlags.ParseJobConfig("fill"),
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
	Name:  "notify",
	Usage: "notify the provided queueing system of gaps to index allowing tipset-workers to perform the indexing.",
	Description: `
The notify command will insert tasks into the provided queueing system for consumption by tipset-workers.
This command should be used when lily is configured to perform distributed indexing.
`,
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
				JobConfig: RunFlags.ParseJobConfig("fill-notify"),
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
