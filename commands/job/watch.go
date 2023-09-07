package job

import (
	"fmt"
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/schedule"
)

type watchOps struct {
	confidence int
	workers    int
	bufferSize int
	interval   int
}

var watchFlags watchOps

var WatchConfidenceFlag = &cli.IntFlag{
	Name:        "confidence",
	Usage:       "Sets the size of the cache used to hold tipsets for possible reversion before being committed to the database.",
	EnvVars:     []string{"LILY_CONFIDENCE"},
	Value:       2,
	Destination: &watchFlags.confidence,
}

var WatchIntervalFlag = &cli.IntFlag{
	Name:        "interval",
	Usage:       "The interval for specific task",
	Value:       120,
	Destination: &watchFlags.interval,
}
var WatchWorkersFlag = &cli.IntFlag{
	Name:        "workers",
	Usage:       "Sets the number of tipsets that may be simultaneous indexed while watching.",
	EnvVars:     []string{"LILY_WATCH_WORKERS"},
	Value:       2,
	Destination: &watchFlags.workers,
}
var WatchBufferSizeFlag = &cli.IntFlag{
	Name:        "buffer-size",
	Usage:       "Set the number of tipsets the watcher will buffer while waiting for a worker to accept the work.",
	EnvVars:     []string{"LILY_WATCH_BUFFER"},
	Value:       5,
	Destination: &watchFlags.bufferSize,
}

//revive:disable
var WatchCmd = &cli.Command{
	Name:  "watch",
	Usage: "watch the head of the filecoin blockchain and index each new head as it becomes available",
	Description: `
The watch command subscribes to incoming tipsets from the filecoin blockchain and indexes them as the arrive.

Since it may be the case that tipsets arrive at a rate greater than their rate of indexing the watch job maintains a
queue of tipsets to index. Consumption of this queue can be configured via the --workers flag. Increasing the value provided
to the --workers flag will allow the watch job to index tipsets simultaneously (Note: this will use a significant amount of system resources).

Since it may be the case that lily experiences a reorg while the watch job is observing the head of the chain
the --confidence flag may be used to buffer the amount of tipsets observed before it begins indexing - illustrated by the below diagram:

             *unshift*        *unshift*      *unshift*       *unshift*
                │  │            │  │            │  │            │  │
             ┌──▼──▼──┐      ┌──▼──▼──┐      ┌──▼──▼──┐      ┌──▼──▼──┐
             │        │      │  ts10  │      │  ts11  │      │  ts12  │
   ...  ---> ├────────┤ ---> ├────────┤ ---> ├────────┤ ---> ├────────┤ --->  ...
             │  ts09  │      │  ts09  │      │  ts10  │      │  ts11  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts08  │      │  ts08  │      │  ts09  │      │  ts10  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ...   │      │  ...   │      │  ...   │      │  ...   │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts02  │      │  ts02  │      │  ts03  │      │  ts04  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts01  │      │  ts01  │      │  ts02  │      │  ts03  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts00  │      │  ts00  │      │  ts01  │      │  ts02  │
             └────────┘      └────────┘      └──│──│──┘      └──│──│──┘
                                                ▼  ▼  *pop*     ▼  ▼  *pop*
                                             ┌────────┐      ┌────────┐
              (confidence=10 :: length=10)   │  ts00  │      │  ts01  │
                                             └────────┘      └────────┘
                                              (process)       (process)

As and example, the below command:
  $ lily job run --tasks-block_header,messages watch --confidence=10 --workers=2
watches the chain head and only indexes a tipset after observing 10 subsequent tipsets indexing at most two tipset simultaneously.
`,
	Flags: []cli.Flag{
		WatchConfidenceFlag,
		WatchWorkersFlag,
		WatchBufferSizeFlag,
		WatchIntervalFlag,
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
		return nil
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
			JobConfig:  RunFlags.ParseJobConfig("watch"),
			BufferSize: watchFlags.bufferSize,
			Confidence: watchFlags.confidence,
			Workers:    watchFlags.workers,
			Interval:   watchFlags.interval,
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

		cfg := &lily.LilyWatchNotifyConfig{
			JobConfig: RunFlags.ParseJobConfig("watch-notify"),

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
