package commands

import (
	"github.com/urfave/cli/v2"

	lotuscli "github.com/filecoin-project/lotus/cli"
)

var StopCmd = &cli.Command{
	Name:  "stop",
	Usage: "Stop a running lily daemon",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
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
