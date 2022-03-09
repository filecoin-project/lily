package job

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	lotuscli "github.com/filecoin-project/lotus/cli"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/lens/client"
	"github.com/filecoin-project/lily/schedule"
)

var log = logging.Logger("lily/commands/job")

var JobCmd = &cli.Command{
	Name:  "job",
	Usage: "Manage jobs being run by the daemon.",
	Flags: []cli.Flag{
		RunApiAddrFlag,
		RunApiTokenFlag,
	},
	Subcommands: []*cli.Command{
		JobCreateCmd,
		JobStartCmd,
		JobStopCmd,
		JobWaitCmd,
		JobListCmd,
	},
}

var JobCreateCmd = &cli.Command{
	Name:  "create",
	Usage: "create and run a job against the daemon",
	Flags: []cli.Flag{
		RunNameFlag,
		RunStorageFlag,
		RunTaskFlag,
		RunExcludeTaskFlag,
		RunWindowFlag,
		RunRestartDelayFlag,
		RunRestartCompletion,
		RunRestartFailure,
	},
	Before: func(cctx *cli.Context) error {
		if RunTaskFlag.IsSet() && RunExcludeTaskFlag.IsSet() {
			return xerrors.Errorf("both %s and % cannot be set, use one or the other", RunExcludeTaskFlag.Name, RunTaskFlag.Name)
		}

		if RunExcludeTaskFlag.IsSet() {
			// build a map of all possible tasks
			tasks := make(map[string]bool)
			for _, task := range indexer.AllTasks {
				tasks[task] = true
			}
			// remove excluded tasks from map
			for _, task := range RunFlags.ExcludeTasks.Value() {
				delete(tasks, task)
			}
			// build a list of the resulting tasks
			setTasks := cli.NewStringSlice()
			for task := range tasks {
				if err := setTasks.Set(task); err != nil {
					return err
				}
			}
			RunFlags.Tasks = setTasks
		}

		return nil
	},
	Subcommands: []*cli.Command{
		WalkCmd,
		WatchCmd,
		IndexCmd,
		GapCmd,
		SurveyCmd,
	},
}

var JobStartCmd = &cli.Command{
	Name:  "start",
	Usage: "start an existing job against the daemon.",
	Action: func(cctx *cli.Context) error {
		var idStr string
		if idStr = cctx.Args().First(); idStr == "" {
			return xerrors.Errorf("job id argument required")
		}
		jobID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return xerrors.Errorf("failed to parse jobID %s: %w", idStr, err)
		}

		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		return api.LilyJobStart(ctx, schedule.JobID(jobID))
	},
}

var JobStopCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop an existing job against the daemon.",
	Action: func(cctx *cli.Context) error {
		var idStr string
		if idStr = cctx.Args().First(); idStr == "" {
			return xerrors.Errorf("job id argument required")
		}
		jobID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return xerrors.Errorf("failed to parse jobID %s: %w", idStr, err)
		}

		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		return api.LilyJobStop(ctx, schedule.JobID(jobID))
	},
}

var JobWaitCmd = &cli.Command{
	Name:  "wait",
	Usage: "wait on an existing job to complete.",
	Action: func(cctx *cli.Context) error {
		var idStr string
		if idStr = cctx.Args().First(); idStr == "" {
			return xerrors.Errorf("job id argument required")
		}
		jobID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return xerrors.Errorf("failed to parse jobID %s: %w", idStr, err)
		}

		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyJobWait(ctx, schedule.JobID(jobID))
		if err != nil {
			return err
		}
		prettyJob, err := json.MarshalIndent(res, "", "\t")
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "%s\n", prettyJob); err != nil {
			return err
		}
		return nil
	},
}

var JobListCmd = &cli.Command{
	Name:  "list",
	Usage: "list all daemon jobs and their status.",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		jobs, err := api.LilyJobList(ctx)
		if err != nil {
			return err
		}
		prettyJobs, err := json.MarshalIndent(jobs, "", "\t")
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stdout, "%s\n", prettyJobs); err != nil {
			return err
		}
		return nil
	},
}
