package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

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
	Flags: flagSet(
		clientAPIFlagSet,
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "tasks",
				Usage:       "Comma separated list of survey tasks to run. Each task is reported separately in the database.",
				Value:       strings.Join([]string{network.PeerAgentsTask}, ","),
				Destination: &surveyFlags.tasks,
			},
			&cli.DurationFlag{
				Name:        "interval",
				Usage:       "Interval to wait between each survey",
				Value:       10 * time.Minute,
				Destination: &surveyFlags.interval,
			},
			&cli.StringFlag{
				Name:        "storage",
				Usage:       "Name of storage that results will be written to.",
				Value:       "",
				Destination: &surveyFlags.storage,
			},
			&cli.StringFlag{
				Name:        "name",
				Usage:       "Name of job for easy identification later.",
				Value:       "",
				Destination: &surveyFlags.name,
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		survName := fmt.Sprintf("survey_%d", time.Now().Unix())
		if surveyFlags.name != "" {
			survName = surveyFlags.name
		}

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		cfg := &lily.LilySurveyConfig{
			Name:                survName,
			Tasks:               strings.Split(surveyFlags.tasks, ","),
			Interval:            surveyFlags.interval,
			RestartDelay:        0,
			RestartOnCompletion: false,
			RestartOnFailure:    true,
			Storage:             surveyFlags.storage,
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
