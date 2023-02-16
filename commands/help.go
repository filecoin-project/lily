package commands

import (
	"github.com/urfave/cli/v2"
)

var HelpCmd = &cli.Command{
	Name:      "help",
	Aliases:   []string{"h"},
	Usage:     "Shows a list of commands or help for one command",
	ArgsUsage: "[command]",
}
