package commands

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/lily"
)

type walkOps struct {
	from    int64
	to      int64
	workers int
}

var walkFlags walkOps

var WalkCmd = &cli.Command{
	Name:  "walk",
	Usage: "Start a daemon job to walk a range of the filecoin blockchain.",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:        "from",
			Usage:       "Limit actor and message processing to tipsets at or above `HEIGHT`",
			EnvVars:     []string{"LILY_FROM"},
			Destination: &walkFlags.from,
			Required:    true,
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			EnvVars:     []string{"LILY_TO"},
			Destination: &walkFlags.to,
			Required:    true,
		},
		&cli.IntFlag{
			Name:        "workers",
			Usage:       "Sets the number of tipsets that may be simultaneous indexed while walking",
			EnvVars:     []string{"LILY_WALK_WORKERS"},
			Value:       1,
			Destination: &walkFlags.workers,
		},
	},
	Before: func(cctx *cli.Context) error {
		from, to := walkFlags.from, walkFlags.to
		if to < from {
			return xerrors.Errorf("value of --to (%d) should be >= --from (%d)", to, from)
		}

		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		cfg := &lily.LilyWalkConfig{
			JobConfig: jobConfigFromFlags(cctx, runFlags),
			From:      walkFlags.from,
			To:        walkFlags.to,
			Workers:   walkFlags.workers,
		}

		api, closer, err := GetAPI(ctx, runFlags.apiAddr, runFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyWalk(ctx, cfg)
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}
		return nil
	},
}
