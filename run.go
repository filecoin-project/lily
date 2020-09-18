package main

import (
	"context"
	"fmt"
	"io"
	_ "net/http/pprof"
	"os"

	_ "github.com/lib/pq"

	logging "github.com/ipfs/go-log/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jconfig "github.com/uber/jaeger-client-go/config"
	"github.com/urfave/cli/v2"

	lens "github.com/filecoin-project/sentinel-visor/lens"
	lotuslens "github.com/filecoin-project/sentinel-visor/lens/lotus"
	vapi "github.com/filecoin-project/sentinel-visor/lens/lotus"
	indexer2 "github.com/filecoin-project/sentinel-visor/services/indexer"
	processor2 "github.com/filecoin-project/sentinel-visor/services/processor"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start visor",
	Subcommands: []*cli.Command{
		processorCmd,
		indexerCmd,
	},
}

var processorCmd = &cli.Command{
	Name:  "processor",
	Usage: "Non-singleton processor of the lotus blockchain",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 10,
		},
	},
	Action: func(cctx *cli.Context) error {
		ll := cctx.String("log-level")
		if err := logging.SetLogLevel("*", ll); err != nil {
			return err
		}

		if cctx.Bool("tracing") {
			closer, err := setupTracing()
			if err != nil {
				log.Errorw("failed to initialize tracing subsystem", "error", err)
				return err
			}
			defer closer.Close()
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return err
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("closing base", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return err
		}

		processor := processor2.NewProcessor(rctx.db, rctx.api)
		if err := processor.InitHandler(ctx, cctx.Int("max-batch")); err != nil {
			return err
		}
		processor.Start(ctx)

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		<-ctx.Done()
		os.Exit(0)
		return nil
	},
}

var indexerCmd = &cli.Command{
	Name:  "indexer",
	Usage: "Singleton indexer of the lotus blockchain",
	Action: func(cctx *cli.Context) error {
		ll := cctx.String("log-level")
		if err := logging.SetLogLevel("*", ll); err != nil {
			return err
		}

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		if cctx.Bool("tracing") {
			closer, err := setupTracing()
			if err != nil {
				log.Errorw("failed to initialize tracing subsystem", "error", err)
				return err
			}
			defer closer.Close()
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return err
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("closing statebase", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return err
		}

		indexer := indexer2.NewIndexer(rctx.db, rctx.api)
		if err := indexer.InitHandler(ctx); err != nil {
			return err
		}

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		// Start the indexer and wait for it to complete or to be cancelled.
		done := make(chan struct{})
		go func() {
			defer close(done)
			// TODO if the lotus daemon hangs up Start will exit. It should restart and wait for lotus to come back online.
			err = indexer.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return nil
		case <-done:
			return err
		}
	},
}

type RunContext struct {
	api    lens.API
	closer lotuslens.APICloser
	db     *storage.Database
}

func setupStorageAndAPI(cctx *cli.Context) (context.Context, *RunContext, error) {
	ctx, api, closer, err := vapi.GetFullNodeAPI(cctx)
	if err != nil {
		return nil, nil, err
	}

	db, err := storage.NewDatabase(ctx, cctx.String("db"))
	if err != nil {
		return nil, nil, err
	}

	return ctx, &RunContext{api, closer, db}, nil
}

func setupTracing() (io.Closer, error) {
	// Read config from env, see https://github.com/jaegertracing/jaeger-client-go#environment-variables
	jaegerConfig, err := jconfig.FromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to get jaeger config from environment: %w", err)
	}

	if jaegerConfig.ServiceName == "" {
		jaegerConfig.ServiceName = "sentinel-visor"
	}

	traceLogger := logging.Logger("tracing")

	// Setup the standard remote reporter based on env vars.
	remoteReporter, err := jaegerConfig.Reporter.NewReporter(jaegerConfig.ServiceName, jaeger.NewNullMetrics(), &jaegerLogger{logger: traceLogger})
	if err != nil {
		return nil, fmt.Errorf("failed to create new jaeger reporter: %w", err)
	}

	// Construct a new tracer.
	tracer, closer, err := jaegerConfig.NewTracer(jconfig.Reporter(remoteReporter))
	if err != nil {
		return nil, fmt.Errorf("failed to create new jager tracer: %w", err)
	}

	opentracing.SetGlobalTracer(tracer)

	return closer, nil
}

type jaegerLogger struct {
	logger logging.EventLogger
}

// Error logs a message at error priority
func (l *jaegerLogger) Error(msg string) {
	l.logger.Error(msg)
}

// Infof logs a message at info priority
func (l *jaegerLogger) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args...)
}
