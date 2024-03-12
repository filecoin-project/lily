package job

import (
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/lens/lily"

	lotuscli "github.com/filecoin-project/lotus/cli"
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
