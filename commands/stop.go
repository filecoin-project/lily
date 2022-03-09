package commands

import (
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/lens/client"
)

var StopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop a running lily daemon",
	Flags: flagSet(
		clientAPIFlagSet,
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := client.GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		err = lapi.Shutdown(ctx)
		if err != nil {
			return err
		}

		return nil
	},
}
