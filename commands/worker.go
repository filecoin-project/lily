package commands

import (
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
)

type workerOpts struct {
	apiAddr  string
	apiToken string
}

var workerFlags workerOpts

var WorkerCmd = &cli.Command{
	Name: "tsworker",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &workerFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &workerFlags.apiToken,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, workerFlags.apiAddr, workerFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		err = api.StartTipSetWorker(ctx)
		if err != nil {
			return err
		}
		return nil
	},
}
