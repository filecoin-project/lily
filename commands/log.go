package commands

import (
	"fmt"
	"regexp"
	"sort"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var LogCmd = &cli.Command{
	Name:  "log",
	Usage: "Manage logging",
	Description: `
	Manage lily logging systems.

	Environment Variables:
	GOLOG_LOG_LEVEL - Default log level for all log systems
	GOLOG_LOG_FMT   - Change output log format (json, nocolor)
	GOLOG_FILE      - Write logs to file
	GOLOG_OUTPUT    - Specify whether to output to file, stderr, stdout or a combination, i.e. file+stderr
`,
	Subcommands: []*cli.Command{
		LogList,
		LogSetLevel,
		LogSetLevelRegex,
	},
}

var LogList = &cli.Command{
	Name:  "list",
	Usage: "List log systems",
	Flags: flagSet(
		clientAPIFlagSet,
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		systems, err := api.LogList(ctx)
		if err != nil {
			return err
		}

		sort.Strings(systems)

		for _, system := range systems {
			fmt.Println(system)
		}

		return nil
	},
}

var LogSetLevel = &cli.Command{
	Name:      "set-level",
	Usage:     "Set log level",
	ArgsUsage: "[level]",
	Description: `Set the log level for logging systems:

   The system flag can be specified multiple times.

   eg) log set-level --system chain --system chainxchg debug

   Available Levels:
   debug
   info
   warn
   error
`,
	Flags: flagSet(
		clientAPIFlagSet,
		[]cli.Flag{
			&cli.StringSliceFlag{
				Name:  "system",
				Usage: "limit to log system",
				Value: &cli.StringSlice{},
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() {
			return fmt.Errorf("level is required")
		}

		systems := cctx.StringSlice("system")
		if len(systems) == 0 {
			var err error
			systems, err = api.LogList(ctx)
			if err != nil {
				return err
			}
		}

		for _, system := range systems {
			if err := api.LogSetLevel(ctx, system, cctx.Args().First()); err != nil {
				return xerrors.Errorf("setting log level on %s: %v", system, err)
			}
		}

		return nil
	},
}

var LogSetLevelRegex = &cli.Command{
	Name:      "set-level-regex",
	Usage:     "Set log level via regular expression",
	ArgsUsage: "[level] [regex]",
	Description: `Set the log level for logging systems via a regular expression matching all logging systems:

   eg) log set-level-regex info 'lily/*'
   eg) log set-level-regex debug 'lily/tasks/*'

   Available Levels:
   debug
   info
   warn
   error
`,
	Flags: flagSet(
		clientAPIFlagSet,
	),
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)

		api, closer, err := GetAPI(ctx, clientAPIFlags.apiAddr, clientAPIFlags.apiToken)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() {
			return fmt.Errorf("level and regular expression are required")
		}

		if cctx.Args().Len() != 2 {
			return fmt.Errorf("extactyl two arguments required [level] [regex]")
		}

		if cctx.Args().First() == "" {
			return fmt.Errorf("level is required")
		}

		if cctx.Args().Get(1) == "" {
			return fmt.Errorf("regex is required")
		}

		if _, err := regexp.Compile(cctx.Args().Get(1)); err != nil {
			return fmt.Errorf("regex does not complie: %w", err)
		}

		if err := api.LogSetLevelRegex(ctx, cctx.Args().Get(1), cctx.Args().First()); err != nil {
			return xerrors.Errorf("setting log level via regex: %w", err)
		}

		return nil
	},
}
