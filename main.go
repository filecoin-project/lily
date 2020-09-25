package main

import (
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/version"
)

var log = logging.Logger("visor")

func main() {
	if err := logging.SetLogLevel("*", "info"); err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:    "visor",
		Usage:   "Filecoin Chain Monitoring Utility",
		Version: version.String(),
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
			&cli.IntFlag{
				Name:    "db-pool-size",
				EnvVars: []string{"LOTUS_DB_POOL_SIZE"},
				Value:   75,
			},
			&cli.StringFlag{
				Name:    "log-level",
				EnvVars: []string{"GOLOG_LOG_LEVEL"},
				Value:   "debug",
				Usage:   "Set the default log level for all loggers to `LEVEL`",
			},
			&cli.StringFlag{
				Name:    "log-level-named",
				EnvVars: []string{"VISOR_LOG_LEVEL_NAMED"},
				Value:   "",
				Usage:   "A comma delimited list of named loggers and log levels formatted as name:level, for example 'logger1:debug,logger2:info'",
			},
			&cli.BoolFlag{
				Name:    "tracing",
				EnvVars: []string{"VISOR_TRACING"},
				Value:   true,
			},
			&cli.StringFlag{
				Name:    "jaeger-agent-host",
				EnvVars: []string{"JAEGER_AGENT_HOST"},
				Value:   "localhost",
			},
			&cli.IntFlag{
				Name:    "jaeger-agent-port",
				EnvVars: []string{"JAEGER_AGENT_PORT"},
				Value:   6831,
			},
			&cli.StringFlag{
				Name:    "jaeger-service-name",
				EnvVars: []string{"JAEGER_SERVICE_NAME"},
				Value:   "sentinel-visor",
			},
			&cli.StringFlag{
				Name:    "jaeger-sampler-type",
				EnvVars: []string{"JAEGER_SAMPLER_TYPE"},
				Value:   "probabilistic",
			},
			&cli.Float64Flag{
				Name:    "jaeger-sampler-param",
				EnvVars: []string{"JAEGER_SAMPLER_PARAM"},
				Value:   0.0001,
			},
		},
		Commands: []*cli.Command{
			processCmd,
			indexCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
