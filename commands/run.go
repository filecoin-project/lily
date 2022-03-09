package commands

import (
	"fmt"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/indexer"
	"github.com/filecoin-project/lily/lens/lily"
)

type runOpts struct {
	apiAddr  string
	apiToken string
	storage  string
	name     string

	tasks        *cli.StringSlice
	excludeTasks *cli.StringSlice

	window       time.Duration
	restartDelay time.Duration

	restartCompletion bool
	restartFailure    bool
}

var runFlags runOpts

func jobConfigFromFlags(cctx *cli.Context, opts runOpts) lily.LilyJobConfig {
	ctx := lotuscli.ReqContext(cctx)
	api, closer, err := GetAPI(ctx, runFlags.apiAddr, runFlags.apiToken)
	if err != nil {
		log.Fatalf("failed to connect to the daemon (is it running?): %s", err)
	}
	defer closer()
	jobID := api.LilyNextJobID()
	if runFlags.name == "" {
		runFlags.name = fmt.Sprintf("job_%d", jobID)
	}
	return lily.LilyJobConfig{
		Name:                opts.name,
		Storage:             opts.storage,
		Tasks:               opts.tasks.Value(),
		Window:              opts.window,
		RestartOnFailure:    opts.restartFailure,
		RestartOnCompletion: opts.restartCompletion,
		RestartDelay:        opts.restartDelay,
	}
}

var runTaskFlag = &cli.StringSliceFlag{
	Name:        "tasks",
	Usage:       "Comma separated list of tasks to run. Each task is reported separately in the database.",
	EnvVars:     []string{"LILY_TASKS"},
	Destination: runFlags.tasks,
}

var runExcludeTaskFlag = &cli.StringSliceFlag{
	Name:        "exclude-tasks",
	Usage:       "Comma separated list of tasks to exclude.",
	EnvVars:     []string{"LILY_EXCLUDE_TASKS"},
	Destination: runFlags.excludeTasks,
}

var runStorageFlag = &cli.StringFlag{
	Name:        "storage",
	Usage:       "Name of storage that results will be written to.",
	EnvVars:     []string{"LILY_STORAGE"},
	Value:       "",
	Destination: &runFlags.storage,
}

var runNameFlag = &cli.StringFlag{
	Name:        "name",
	Usage:       "Name of job for easy identification later.",
	EnvVars:     []string{"LILY_JOB_NAME"},
	Value:       "",
	Destination: &runFlags.name,
}

var runWindowFlag = &cli.DurationFlag{
	Name:        "window",
	Usage:       "Duration after which any indexing work not completed will be marked incomplete",
	EnvVars:     []string{"LILY_WINDOW"},
	Value:       builtin.EpochDurationSeconds * time.Second,
	Destination: &runFlags.window,
}
var runApiAddrFlag = &cli.StringFlag{
	Name:        "api",
	Usage:       "Address of lily api in multiaddr format.",
	EnvVars:     []string{"LILY_API"},
	Value:       "/ip4/127.0.0.1/tcp/1234",
	Destination: &runFlags.apiAddr,
}

var runApiTokenFlag = &cli.StringFlag{
	Name:        "api-token",
	Usage:       "Authentication token for lily api.",
	EnvVars:     []string{"LILY_API_TOKEN"},
	Value:       "",
	Destination: &runFlags.apiToken,
}

var runRestartDelayFlag = &cli.DurationFlag{
	Name:        "restart-delay",
	Usage:       "Duration to wait before restarting job",
	EnvVars:     []string{"LILY_RESTART_DELAY"},
	Value:       0,
	Destination: &runFlags.restartDelay,
}

var runRestartCompletion = &cli.BoolFlag{
	Name:        "restart-on-completion",
	Usage:       "Restart the job after it completes",
	EnvVars:     []string{"LILY_RESTART_COMPLETION"},
	Value:       false,
	Destination: &runFlags.restartCompletion,
}

var runRestartFailure = &cli.BoolFlag{
	Name:        "restart-on-failure",
	Usage:       "Restart the job if it fails",
	EnvVars:     []string{"LILY_RESTART_FAILURE"},
	Value:       false,
	Destination: &runFlags.restartFailure,
}

var RunCmd = &cli.Command{
	Name:  "run",
	Usage: "Run a job against the daemon",
	Flags: []cli.Flag{
		runApiAddrFlag,
		runApiTokenFlag,
		runNameFlag,
		runStorageFlag,
		runTaskFlag,
		runExcludeTaskFlag,
		runWindowFlag,
		runRestartDelayFlag,
		runRestartCompletion,
		runRestartFailure,
	},
	Before: func(cctx *cli.Context) error {
		if runTaskFlag.IsSet() && runExcludeTaskFlag.IsSet() {
			return xerrors.Errorf("both %s and % cannot be set, use one or the other", runExcludeTaskFlag.Name, runTaskFlag.Name)
		}

		if runExcludeTaskFlag.IsSet() {
			// build a map of all possible tasks
			tasks := make(map[string]bool)
			for _, task := range indexer.AllTasks {
				tasks[task] = true
			}
			// remove excluded tasks from map
			for _, task := range runFlags.excludeTasks.Value() {
				delete(tasks, task)
			}
			// build a list of the resulting tasks
			setTasks := cli.NewStringSlice()
			for task := range tasks {
				if err := setTasks.Set(task); err != nil {
					return err
				}
			}
			runFlags.tasks = setTasks
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
