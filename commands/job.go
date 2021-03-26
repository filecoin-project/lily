package commands

import (
	"encoding/json"
	"fmt"
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/schedule"
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

type jobControlOpts struct {
	ID int
}

var jobControlFlag jobControlOpts

var JobStartCmd = &cli.Command{
	Name:   "start",
	Usage:  "start a job.",
	Before: initialize,
	After:  destroy,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "ID",
			Usage:       "ID of job to start",
			Required:    true,
			Destination: &jobControlFlag.ID,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		return lilyAPI.LilyJobStart(ctx, schedule.JobID(jobControlFlag.ID))
	},
}

var JobStopCmd = &cli.Command{
	Name:   "stop",
	Usage:  "stop a job.",
	Before: initialize,
	After:  destroy,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "ID",
			Usage:       "ID of job to stop",
			Required:    true,
			Destination: &jobControlFlag.ID,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		return lilyAPI.LilyJobStop(ctx, schedule.JobID(jobControlFlag.ID))
	},
}

var JobListCmd = &cli.Command{
	Name:   "list",
	Usage:  "list all jobs and their status",
	Before: initialize,
	After:  destroy,
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		jobs, err := lilyAPI.LilyJobList(ctx)
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
