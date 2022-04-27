package job

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/lily"
)

type runOpts struct {
	Storage string
	Name    string

	Tasks *cli.StringSlice

	Window       time.Duration
	RestartDelay time.Duration

	RestartCompletion bool
	RestartFailure    bool
}

func (r runOpts) ParseJobConfig(kind string) lily.LilyJobConfig {
	if RunFlags.Name == "" {
		RunFlags.Name = fmt.Sprintf("%s_%d", kind, time.Now().Unix())
	}
	if len(RunFlags.Tasks.Value()) == 0 {
		// TODO don't panic
		panic("need tasks")
	}
	// TODO handle task wild card *
	return lily.LilyJobConfig{
		Name:                RunFlags.Name,
		Storage:             RunFlags.Storage,
		Tasks:               RunFlags.Tasks.Value(),
		Window:              RunFlags.Window,
		RestartOnFailure:    RunFlags.RestartFailure,
		RestartOnCompletion: RunFlags.RestartCompletion,
		RestartDelay:        RunFlags.RestartDelay,
	}
}

var RunFlags runOpts

var RunWindowFlag = &cli.DurationFlag{
	Name:        "window",
	Usage:       "Duaration after which job execution will be canceled",
	EnvVars:     []string{"LILY_JOB_WINDOW"},
	Value:       0,
	Destination: &RunFlags.Window,
}

var RunTaskFlag = &cli.StringSliceFlag{
	Name:        "tasks",
	Usage:       "Comma separated list of tasks to run in job. Each task is reported separately in the storage backend.",
	EnvVars:     []string{"LILY_JOB_TASKS"},
	Destination: RunFlags.Tasks,
}

var RunStorageFlag = &cli.StringFlag{
	Name:        "storage",
	Usage:       "Name of storage backend the job will write result to.",
	EnvVars:     []string{"LILY_JOB_STORAGE"},
	Value:       "",
	Destination: &RunFlags.Storage,
}

var RunNameFlag = &cli.StringFlag{
	Name:        "name",
	Usage:       "Name of job for easy identification later. Will appear as 'reporter' in the visor_processing_reports table.",
	EnvVars:     []string{"LILY_JOB_NAME"},
	Value:       "",
	Destination: &RunFlags.Name,
}

var RunRestartDelayFlag = &cli.DurationFlag{
	Name:        "restart-delay",
	Usage:       "Duration to wait before restarting job after it ends execution",
	EnvVars:     []string{"LILY_JOB_RESTART_DELAY"},
	Value:       0,
	Destination: &RunFlags.RestartDelay,
}

var RunRestartCompletion = &cli.BoolFlag{
	Name:        "restart-on-completion",
	Usage:       "Restart the job after it completes.",
	EnvVars:     []string{"LILY_JOB_RESTART_COMPLETION"},
	Value:       false,
	Destination: &RunFlags.RestartCompletion,
}

var RunRestartFailure = &cli.BoolFlag{
	Name:        "restart-on-failure",
	Usage:       "Restart the job if it fails.",
	EnvVars:     []string{"LILY_JOB_RESTART_FAILURE"},
	Value:       false,
	Destination: &RunFlags.RestartFailure,
}

type notifyOps struct {
	queue string
}

var notifyFlags notifyOps

var NotifyQueueFlag = &cli.StringFlag{
	Name:        "queue",
	Usage:       "Name of queue system that job will notify.",
	EnvVars:     []string{"LILY_JOB_QUEUE"},
	Value:       "",
	Destination: &notifyFlags.queue,
}

type rangeOps struct {
	from int64
	to   int64
}

var rangeFlags rangeOps

func (r rangeOps) validate() error {
	from, to := rangeFlags.from, rangeFlags.to
	if to < from {
		return xerrors.Errorf("value of --to (%d) should be >= --from (%d)", to, from)
	}

	return nil
}

var RangeFromFlag = &cli.Int64Flag{
	Name:        "from",
	Usage:       "Limit actor and message processing to tipsets at or above `HEIGHT`",
	EnvVars:     []string{"LILY_FROM"},
	Destination: &rangeFlags.from,
	Required:    true,
}

var RangeToFlag = &cli.Int64Flag{
	Name:        "to",
	Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
	EnvVars:     []string{"LILY_TO"},
	Destination: &rangeFlags.to,
	Required:    true,
}
