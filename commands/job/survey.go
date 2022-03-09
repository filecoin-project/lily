package job

import (
	"os"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/lens/client"
	"github.com/filecoin-project/lily/lens/lily"
)

var surveyFlags struct {
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

		api, closer, err := client.GetAPI(ctx, RunFlags.ApiAddr, RunFlags.ApiToken)
		if err != nil {
			return err
		}
		defer closer()

		cfg := &lily.LilySurveyConfig{
			JobConfig: JobConfigFromFlags(cctx, RunFlags),
			Interval:  surveyFlags.interval,
		}

		res, err := api.LilySurvey(ctx, cfg)
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}
		return nil
	},
}
