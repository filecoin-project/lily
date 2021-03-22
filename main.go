package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/sentinel-visor/commands"
	"github.com/filecoin-project/sentinel-visor/version"
)

var log = logging.Logger("visor")

func main() {
	// Set up a context that is canceled when the command is interrupted
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up a signal handler to cancel the context
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := logging.SetLogLevel("*", "info"); err != nil {
		log.Fatal(err)
	}

	defaultName := "visor_" + version.String()
	hostname, err := os.Hostname()
	if err == nil {
		defaultName = fmt.Sprintf("%s_%s_%d", defaultName, hostname, os.Getpid())
	}

	app := &cli.App{
		Name:    "visor",
		Usage:   "Filecoin Chain Monitoring Utility",
		Version: version.String(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "lens",
				EnvVars: []string{"VISOR_LENS"},
				Value:   "lotus",
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
			&cli.BoolFlag{
				Name:    "repo-read-only",
				EnvVars: []string{"VISOR_REPO_READ_ONLY"},
				Value:   true,
				Usage:   "Open the repo in read only mode",
			},
			&cli.StringFlag{
				Name:    "api",
				EnvVars: []string{"FULLNODE_API_INFO"},
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "db",
				EnvVars: []string{"LOTUS_DB"},
				Value:   "",
				Usage:   "A connection string for the postgres database, for example postgres://postgres:password@localhost:5432/postgres",
			},
			&cli.IntFlag{
				Name:    "db-pool-size",
				EnvVars: []string{"LOTUS_DB_POOL_SIZE"},
				Value:   75,
			},
			&cli.BoolFlag{
				Name:    "db-allow-upsert",
				EnvVars: []string{"LOTUS_DB_ALLOW_UPSERT"},
				Value:   false,
			},
			&cli.IntFlag{
				Name:    "lens-cache-hint",
				EnvVars: []string{"VISOR_LENS_CACHE_HINT"},
				Value:   1024 * 1024,
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
			&cli.StringFlag{
				Name:    "name",
				EnvVars: []string{"VISOR_NAME"},
				Value:   defaultName,
				Usage:   "A name that helps to identify this instance of visor.",
			},
			&cli.BoolFlag{
				Name:    "tracing",
				EnvVars: []string{"VISOR_TRACING"},
				Value:   false,
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
				Value:   "visor",
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
			&cli.BoolFlag{
				Name:    "allow-schema-migration",
				EnvVars: []string{"VISOR_ALLOW_SCHEMA_MIGRATION"},
				Value:   false,
			},
			&cli.StringFlag{
				Name:    "prometheus-port",
				EnvVars: []string{"VISOR_PROMETHEUS_PORT"},
				Value:   ":9991",
			},
			&cli.StringFlag{
				Name:    "lens-postgres-namespace",
				EnvVars: []string{"VISOR_POSTGRES_NAMESPACE"},
				Value:   "main", // we need *some* namespace specified, otherwise GetFilTipSetHead() can't work
				Usage:   "Namespace consulted for current chain head and recency records",
			},
			&cli.BoolFlag{
				Name:    "lens-postgres-preload-recents",
				EnvVars: []string{"VISOR_POSTGRES_PRELOAD_RECENTS"},
				Value:   false,
				Usage:   "List recent reads within selected namespace, and preload as much as possible into the LRU",
			},
		},
		Commands: []*cli.Command{
			commands.Migrate,
			commands.Vector,
			commands.Walk,
			commands.Watch,
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
