package job

import (
	"fmt"
	"os"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/network"
)

var surveyFlags struct {
	tasks    string
	storage  string
	name     string
	interval time.Duration
}

var SurveyCmd = &cli.Command{
	Name:  "survey",
	Usage: "Start a daemon job to survey the node and its environment.",
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name:        "interval",
			Usage:       "Interval to wait between each survey",
			Value:       10 * time.Minute,
			Destination: &surveyFlags.interval,
		},
	},
	Before: func(cctx *cli.Context) error {
		tasks := RunFlags.Tasks.Value()
		if len(tasks) != 1 {
			return fmt.Errorf("survey accepts single task type: '%s'", network.PeerAgentsTask)
		}
		if tasks[0] != network.PeerAgentsTask {
			return fmt.Errorf("unknown task: %s", tasks[0])
		}
		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := commands.GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilySurvey(ctx, &lily.LilySurveyConfig{
			JobConfig: RunFlags.ParseJobConfig("survey"),
			Interval:  surveyFlags.interval,
		})
		if err != nil {
			return err
		}

		return commands.PrintNewJob(os.Stdout, res)
	},
}
