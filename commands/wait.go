package commands

import (
	"context"
	"fmt"
	"time"

	lotuscli "github.com/filecoin-project/lotus/cli"

	"github.com/urfave/cli/v2"
)

var WaitApiCmd = &cli.Command{
	Name:  "wait-api",
	Usage: "Wait for lily api to come online",
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name:  "timeout",
			Usage: "Time to wait for API to become ready",
			Value: 30 * time.Second,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		var timeout <-chan time.Time

		if cctx.Duration("timeout") > 0 {
			timeout = time.NewTimer(cctx.Duration("timeout")).C
		}

		for {
			err := checkAPI(ctx, ClientAPIFlags.ApiAddr, ClientAPIFlags.ApiToken)
			if err == nil {
				return nil
			}
			log.Warnf("API not online yet... (%s)", err)

			select {
			case <-ctx.Done():
				return nil
			case <-timeout:
				return fmt.Errorf("timed out waiting for api to come online")
			case <-time.After(time.Second):
			}
		}
	},
}

func checkAPI(ctx context.Context, addrStr string, token string) error {
	lapi, closer, err := GetAPI(ctx)
	if err != nil {
		return err
	}
	defer closer()

	_, err = lapi.ID(ctx)
	if err != nil {
		return err
	}

	return nil
}
