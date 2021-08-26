package commands

import (
	"encoding/json"
	"fmt"
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/schedule"
)

var JobCmd = &cli.Command{
	Name:  "job",
	Usage: "Manage jobs being run by the daemon.",
	Subcommands: []*cli.Command{
		JobStartCmd,
		JobStopCmd,
		JobListCmd,
	},
}

var jobControlFlags struct {
	ID int
}

var JobStartCmd = &cli.Command{
	Name:  "start",
	Usage: "start a job.",
	Flags: flagSet(
		clientAPIFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "id",
				Usage:       "Identifier of job to start",
				Required:    true,
				Destination: &jobControlFlags.ID,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
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
	Flags: flagSet(
		clientAPIFlagSet,
		[]cli.Flag{
			&cli.IntFlag{
				Name:        "id",
				Usage:       "Identifier of job to stop",
				Required:    true,
				Destination: &jobControlFlags.ID,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
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
	Flags: flagSet(
		clientAPIFlagSet,
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
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
		if _, err := fmt.Fprintf(os.Stdout, "List Jobs:\n%s\n", prettyJobs); err != nil {
			return err
		}
		return nil
	},
}
