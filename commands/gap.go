package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/lily/chain"
	"github.com/filecoin-project/lily/lens/lily"
	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
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
	Subcommands: []*cli.Command{
		GapFillCmd,
		GapFindCmd,
	},
}

var GapFillCmd = &cli.Command{
	Name:  "fill",
	Usage: "Fill gaps in the database",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &gapFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &gapFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			EnvVars:     []string{"LILY_STORAGE"},
			Value:       "",
			Destination: &gapFlags.storage,
		},
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to fill. Each task is reported separately in the database. If empty all task will be filled.",
			EnvVars:     []string{"LILY_TASKS"},
			Value:       "",
			Destination: &gapFlags.tasks,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			EnvVars:     []string{"LILY_JOB_NAME"},
			Value:       "",
			Destination: &gapFlags.name,
		},
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
	Before: func(cctx *cli.Context) error {
		from, to := gapFlags.from, gapFlags.to
		if to < from {
			xerrors.Errorf("value of --to (%d) should be >= --from (%d)", to, from)
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

		var tasks []string
		if gapFlags.tasks == "" {
			tasks = chain.AllTasks
		} else {
			tasks = strings.Split(gapFlags.tasks, ",")
		}

		fillName := fmt.Sprintf("fill_%d", time.Now().Unix())
		if gapFlags.name != "" {
			fillName = gapFlags.name
		}

		res, err := api.LilyGapFill(ctx, &lily.LilyGapFillConfig{
			RestartOnFailure:    false,
			RestartOnCompletion: false,
			RestartDelay:        0,
			Storage:             gapFlags.storage,
			Name:                fillName,
			Tasks:               tasks,
			To:                  gapFlags.to,
			From:                gapFlags.from,
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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "api",
			Usage:       "Address of lily api in multiaddr format.",
			EnvVars:     []string{"LILY_API"},
			Value:       "/ip4/127.0.0.1/tcp/1234",
			Destination: &gapFlags.apiAddr,
		},
		&cli.StringFlag{
			Name:        "api-token",
			Usage:       "Authentication token for lily api.",
			EnvVars:     []string{"LILY_API_TOKEN"},
			Value:       "",
			Destination: &gapFlags.apiToken,
		},
		&cli.StringFlag{
			Name:        "storage",
			Usage:       "Name of storage that results will be written to.",
			Value:       "",
			Destination: &gapFlags.storage,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "Name of job for easy identification later.",
			Value:       "",
			Destination: &gapFlags.name,
		},
		&cli.StringFlag{
			Name:        "tasks",
			Usage:       "Comma separated list of tasks to fill. Each task is reported separately in the database. If empty all task will be filled.",
			Value:       "",
			Destination: &gapFlags.tasks,
		},
		&cli.Uint64Flag{
			Name:        "to",
			Usage:       "to epoch to search for gaps in",
			Destination: &gapFlags.to,
			Required:    true,
		},
		&cli.Uint64Flag{
			Name:        "from",
			Usage:       "from epoch to search for gaps in",
			Destination: &gapFlags.from,
			Required:    true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, gapFlags.apiAddr, gapFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		findName := fmt.Sprintf("find_%d", time.Now().Unix())
		if gapFlags.name != "" {
			findName = gapFlags.name
		}

		var tasks []string
		if gapFlags.tasks == "" {
			tasks = chain.AllTasks
		} else {
			tasks = strings.Split(gapFlags.tasks, ",")
		}

		res, err := api.LilyGapFind(ctx, &lily.LilyGapFindConfig{
			RestartOnFailure:    false,
			RestartOnCompletion: false,
			RestartDelay:        0,
			Storage:             gapFlags.storage,
			Tasks:               tasks,
			Name:                findName,
			To:                  gapFlags.to,
			From:                gapFlags.from,
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
