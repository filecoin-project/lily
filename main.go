package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("visor")

func main() {
	if err := logging.SetLogLevel("*", "info"); err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:    "visor",
		Usage:   "Filecoin Chain Monitoring Utility",
		Version: "v0.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "api",
				EnvVars: []string{"FULLNODE_API_INFO"},
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "db",
				EnvVars: []string{"LOTUS_DB"},
				Value:   "postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
			},
			&cli.StringFlag{
				Name:    "log-level",
				EnvVars: []string{"GOLOG_LOG_LEVEL"},
				Value:   "debug",
			},
		},
		Commands: []*cli.Command{
			runCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
