package job

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens/client"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/schedule"
)

type runOpts struct {
	ApiAddr  string
	ApiToken string
	Storage  string
	name     string

	Tasks        *cli.StringSlice
	ExcludeTasks *cli.StringSlice

	Window       time.Duration
	RestartDelay time.Duration

	RestartCompletion bool
	RestartFailure    bool
}

var RunFlags runOpts

func JobConfigFromFlags(cctx *cli.Context, opts runOpts) lily.LilyJobConfig {
	ctx := lotuscli.ReqContext(cctx)
	api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
	if err != nil {
		log.Fatalf("failed to connect to the daemon (is it running?): %s", err)
	}
	defer closer()
	jobID := api.LilyNextJobID()
	if RunFlags.name == "" {
		RunFlags.name = fmt.Sprintf("job_%d", jobID)
	}
	return lily.LilyJobConfig{
		Name:                opts.name,
		Storage:             opts.Storage,
		Tasks:               opts.Tasks.Value(),
		Window:              opts.Window,
		RestartOnFailure:    opts.RestartFailure,
		RestartOnCompletion: opts.RestartCompletion,
		RestartDelay:        opts.RestartDelay,
	}
}

func printNewJob(w io.Writer, res *schedule.JobSubmitResult) error {
	prettyJob, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", prettyJob); err != nil {
		return err
	}
	return nil
}

var RunTaskFlag = &cli.StringSliceFlag{
	Name:        "tasks",
	Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
	EnvVars:     []string{"LILY_TASKS"},
	Destination: RunFlags.Tasks,
}

var RunExcludeTaskFlag = &cli.StringSliceFlag{
	Name:        "exclude-tasks",
	Usage:       "Comma separated list of tasks to exclude.",
	EnvVars:     []string{"LILY_EXCLUDE_TASKS"},
	Destination: RunFlags.ExcludeTasks,
}

var RunStorageFlag = &cli.StringFlag{
	Name:        "storage",
	Usage:       "Name of storage that results will be written to.",
	EnvVars:     []string{"LILY_STORAGE"},
	Value:       "",
	Destination: &RunFlags.Storage,
}

var RunNameFlag = &cli.StringFlag{
	Name:        "name",
	Usage:       "Name of job for easy identification later.",
	EnvVars:     []string{"LILY_JOB_NAME"},
	Value:       "",
	Destination: &RunFlags.name,
}

var RunWindowFlag = &cli.DurationFlag{
	Name:        "window",
	Usage:       "Duration after which any indexing work not completed will be marked incomplete",
	EnvVars:     []string{"LILY_WINDOW"},
	Value:       builtin.EpochDurationSeconds * time.Second,
	Destination: &RunFlags.Window,
}
var RunApiAddrFlag = &cli.StringFlag{
	Name:        "api",
	Usage:       "Address of lily api in multiaddr format.",
	EnvVars:     []string{"LILY_API"},
	Value:       "/ip4/127.0.0.1/tcp/1234",
	Destination: &RunFlags.ApiAddr,
}

var RunApiTokenFlag = &cli.StringFlag{
	Name:        "api-token",
	Usage:       "Authentication token for lily api.",
	EnvVars:     []string{"LILY_API_TOKEN"},
	Value:       "",
	Destination: &RunFlags.ApiToken,
}

var RunRestartDelayFlag = &cli.DurationFlag{
	Name:        "restart-delay",
	Usage:       "Duration to wait before restarting job",
	EnvVars:     []string{"LILY_RESTART_DELAY"},
	Value:       0,
	Destination: &RunFlags.RestartDelay,
}

var RunRestartCompletion = &cli.BoolFlag{
	Name:        "restart-on-completion",
	Usage:       "Restart the job after it completes",
	EnvVars:     []string{"LILY_RESTART_COMPLETION"},
	Value:       false,
	Destination: &RunFlags.RestartCompletion,
}

var RunRestartFailure = &cli.BoolFlag{
	Name:        "restart-on-failure",
	Usage:       "Restart the job if it fails",
	EnvVars:     []string{"LILY_RESTART_FAILURE"},
	Value:       false,
	Destination: &RunFlags.RestartFailure,
}
