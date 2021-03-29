package commands

import (
	"github.com/urfave/cli/v2"
)

var RunCmd = &cli.Command{
	Name:  "run",
	Usage: "Run a single job without starting a daemon.",
	Subcommands: []*cli.Command{
		RunWatchCmd,
		RunWalkCmd,
	},
}
