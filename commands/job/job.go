package job

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/schedule"

	lotuscli "github.com/filecoin-project/lotus/cli"
)

var JobCmd = &cli.Command{
	Name:  "job",
	Usage: "Manage jobs being run by the daemon.",
	Subcommands: []*cli.Command{
		JobRunCmd,
		JobStartCmd,
		JobStopCmd,
		JobWaitCmd,
		JobListCmd,
	},
}

var JobRunCmd = &cli.Command{
	Name:  "run",
	Usage: "run a job",
	Flags: []cli.Flag{
		RunWindowFlag,
		RunTaskFlag,
		RunStorageFlag,
		RunNameFlag,
		RunRestartDelayFlag,
		RunRestartFailure,
		RunRestartCompletion,
		StopOnError,
	},
	Subcommands: []*cli.Command{
		WalkCmd,
		WatchCmd,
		IndexCmd,
		SurveyCmd,
		GapFillCmd,
		GapFindCmd,
		TipSetWorkerCmd,
	},
}

var jobControlFlags struct {
	ID int
}

var JobStartCmd = &cli.Command{
	Name:  "start",
	Usage: "start a job.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "id",
			Usage:       "Identifier of job to start",
			Required:    true,
			Destination: &jobControlFlags.ID,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		return api.LilyJobStart(ctx, schedule.JobID(jobControlFlags.ID))
	},
}

var JobStopCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop a job.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "id",
			Usage:       "Identifier of job to stop",
			Required:    true,
			Destination: &jobControlFlags.ID,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		return api.LilyJobStop(ctx, schedule.JobID(jobControlFlags.ID))
	},
}

var JobListCmd = &cli.Command{
	Name:  "list",
	Usage: "list all jobs and their status",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := commands.GetAPI(ctx)
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

var JobWaitCmd = &cli.Command{
	Name:  "wait",
	Usage: "wait on a job to complete.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "id",
			Usage:       "Identifier of job to wait on",
			Required:    true,
			Destination: &jobControlFlags.ID,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyJobWait(ctx, schedule.JobID(jobControlFlags.ID))
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
