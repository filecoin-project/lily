package job

import (
	"fmt"
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
)

type walkOps struct {
	interval int `zap:"interval"`
}

var walkFlags walkOps

var WalkIntervalFlag = &cli.IntFlag{
	Name:        "interval",
	Usage:       "The interval for specific task",
	Value:       120,
	Destination: &walkFlags.interval,
}

var WalkCmd = &cli.Command{
	Name:  "walk",
	Usage: "walk and index a range of the filecoin blockchain.",
	Description: `
The walk command will index state based on the list of tasks (--tasks) provided over the specified range (--from --to).
Each epoch will be indexed serially, starting from the heaviest tipset at the upper height (--to) to the lower height (--from).

As and example, the below command:
  $ lily job run --tasks=block_header,messages walk --from=10 --to=20
walks epochs 20 through 10 (inclusive) executing the block_header and messages task for each epoch.
The status of each epoch and its set of tasks can be observed in the visor_processing_reports table.
`,
	Flags: []cli.Flag{
		RangeFromFlag,
		RangeToFlag,
		WalkIntervalFlag,
	},
	Subcommands: []*cli.Command{
		WalkNotifyCmd,
	},
	Before: func(cctx *cli.Context) error {
		tasks := RunFlags.Tasks.Value()
		for _, taskName := range tasks {
			if _, found := tasktype.TaskLookup[taskName]; found {
				continue
			} else if _, found := tasktype.TableLookup[taskName]; found {
				continue
			} else {
				return fmt.Errorf("unknown task: %s", taskName)
			}
		}
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
			JobConfig: RunFlags.ParseJobConfig("walk"),
			From:      rangeFlags.from,
			To:        rangeFlags.to,
			Interval:  walkFlags.interval,
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
	Name:  "notify",
	Usage: "notify the provided queueing system of epochs to index allowing tipset-workers to perform the indexing.",
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

		cfg := &lily.LilyWalkNotifyConfig{
			WalkConfig: lily.LilyWalkConfig{
				JobConfig: RunFlags.ParseJobConfig("walk-notify"),
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
