package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/node/repo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/version"
)

var log = logging.Logger("lily/main")

type UpSchedule struct {
	Height    int64
	Network   uint
	Expensive bool
}

func (u *UpSchedule) String() string {
	return fmt.Sprintf("Height: %d, Network: %d, Expensive: %t", u.Height, u.Network, u.Expensive)
}

type UpScheduleList []*UpSchedule

func (ul UpScheduleList) String() string {
	var sb strings.Builder
	for _, u := range ul {
		sb.WriteString(fmt.Sprintln("\t\t" + u.String()))
	}
	return sb.String()
}

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

	var up UpScheduleList
	for _, u := range filcns.DefaultUpgradeSchedule() {
		up = append(up, &UpSchedule{
			Height:    int64(u.Height),
			Network:   uint(u.Network),
			Expensive: false,
		})
	}

	cli.AppHelpTemplate = commands.AppHelpTemplate

	app := &cli.App{
		Name:    "lily",
		Usage:   "a tool for capturing on-chain state from the filecoin network",
		Version: fmt.Sprintf("VisorVersion: \t%s\n   NewestNetworkVersion: \t%d\n   GenesisFile: \t%s\n   DevNet: \t%t\n   UserVersion: \t%s\n   UpgradeSchedule: \n%s", version.String(), build.NewestNetworkVersion, build.GenesisFile, build.Devnet, build.UserVersion(), up.String()),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				EnvVars:     []string{"GOLOG_LOG_LEVEL"},
				Value:       "info",
				Usage:       "Set the default log level for all loggers to `LEVEL`",
				Destination: &commands.VisorLogFlags.LogLevel,
			},
			&cli.StringFlag{
				Name:        "log-level-named",
				EnvVars:     []string{"LILY_LOG_LEVEL_NAMED"},
				Value:       "",
				Usage:       "A comma delimited list of named loggers and log levels formatted as name:level, for example 'logger1:debug,logger2:info'",
				Destination: &commands.VisorLogFlags.LogLevelNamed,
			},
			&cli.BoolFlag{
				Name:        "jaeger-tracing",
				EnvVars:     []string{"LILY_JAEGER_TRACING"},
				Value:       false,
				Destination: &commands.VisorTracingFlags.Enabled,
			},
			&cli.StringFlag{
				Name:        "jaeger-service-name",
				EnvVars:     []string{"LILY_JAEGER_SERVICE_NAME"},
				Value:       "lily",
				Destination: &commands.VisorTracingFlags.ServiceName,
			},
			&cli.StringFlag{
				Name:        "jaeger-provider-url",
				EnvVars:     []string{"LILY_JAEGER_PROVIDER_URL"},
				Value:       "http://localhost:14268/api/traces",
				Destination: &commands.VisorTracingFlags.ProviderURL,
			},
			&cli.Float64Flag{
				Name:        "jaeger-sampler-ratio",
				EnvVars:     []string{"LILY_JAEGER_SAMPLER_RATIO"},
				Usage:       "If less than 1 probabilistic metrics will be used.",
				Value:       1,
				Destination: &commands.VisorTracingFlags.JaegerSamplerParam,
			},
			&cli.StringFlag{
				Name:        "prometheus-port",
				EnvVars:     []string{"LILY_PROMETHEUS_PORT"},
				Value:       ":9991",
				Destination: &commands.VisorMetricFlags.PrometheusPort,
			},
		},
		HideHelp: true,
		Metadata: commands.Metadata(),
		Commands: []*cli.Command{
			commands.ChainCmd,
			commands.DaemonCmd,
			commands.ExportChainCmd,
			commands.GapCmd,
			commands.HelpCmd,
			commands.InitCmd,
			commands.JobCmd,
			commands.LogCmd,
			commands.MigrateCmd,
			commands.NetCmd,
			commands.SurveyCmd,
			commands.StopCmd,
			commands.SyncCmd,
			commands.WaitApiCmd,
			commands.WalkCmd,
			commands.WatchCmd,
		},
	}
	app.Setup()
	app.Metadata["repoType"] = repo.FullNode
	app.Metadata["traceContext"] = ctx

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}
