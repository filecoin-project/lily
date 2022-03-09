package commands

import (
	"os"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/lens/lily"
)

type gapOps struct {
	apiAddr  string
	apiToken string
	storage  string
	tasks    string
	name     string
	from     uint64
	to       uint64
}

var gapFlags gapOps

var GapCmd = &cli.Command{
	Name:  "gap",
	Usage: "Launch gap filling and finding jobs",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:        "to",
			Usage:       "to epoch to search for gaps in",
			EnvVars:     []string{"LILY_TO"},
			Destination: &gapFlags.to,
			Required:    true,
		},
		&cli.Uint64Flag{
			Name:        "from",
			Usage:       "from epoch to search for gaps in",
			EnvVars:     []string{"LILY_FROM"},
			Destination: &gapFlags.from,
			Required:    true,
		},
	},
	Subcommands: []*cli.Command{
		GapFillCmd,
		GapFindCmd,
	},
}

var GapFillCmd = &cli.Command{
	Name:  "fill",
	Usage: "Fill gaps in the database",
	Before: func(cctx *cli.Context) error {
		from, to := gapFlags.from, gapFlags.to
		if to < from {
			return xerrors.Errorf("value of --to (%d) should be >= --from (%d)", to, from)
		}

		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, gapFlags.apiAddr, gapFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyGapFill(ctx, &lily.LilyGapFillConfig{
			JobConfig: jobConfigFromFlags(cctx, runFlags),
			To:        gapFlags.to,
			From:      gapFlags.from,
		})
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}
		return nil
	},
}

var GapFindCmd = &cli.Command{
	Name:  "find",
	Usage: "find gaps in the database",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, gapFlags.apiAddr, gapFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		res, err := api.LilyGapFind(ctx, &lily.LilyGapFindConfig{
			JobConfig: jobConfigFromFlags(cctx, runFlags),
			To:        gapFlags.to,
			From:      gapFlags.from,
		})
		if err != nil {
			return err
		}
		if err := printNewJob(os.Stdout, res); err != nil {
			return err
		}
		return nil
	},
}
