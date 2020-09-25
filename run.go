package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
	"strings"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/lib/pq"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/xerrors"

	lens "github.com/filecoin-project/sentinel-visor/lens"
	lotuslens "github.com/filecoin-project/sentinel-visor/lens/lotus"
	vapi "github.com/filecoin-project/sentinel-visor/lens/lotus"
	"github.com/filecoin-project/sentinel-visor/metrics"
	indexer2 "github.com/filecoin-project/sentinel-visor/services/indexer"
	processor2 "github.com/filecoin-project/sentinel-visor/services/processor"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var processCmd = &cli.Command{
	Name:  "process",
	Usage: "Process indexed blocks of the lotus blockchain",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 10,
		},
	},
	Action: func(cctx *cli.Context) error {
		err := setupLogging(cctx)
		if err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		if err := setupMetrics(); err != nil {
			return xerrors.Errorf("setup metrics: %w", err)
		}

		if cctx.Bool("tracing") {
			jcfg, err := jaegerConfigFromCliContext(cctx)
			if err != nil {
				return xerrors.Errorf("read jeager config: %w", err)
			}
			closer, err := setupTracing(jcfg)
			if err != nil {
				return xerrors.Errorf("initialize tracing subsystem: %w", err)
			}
			defer closer()
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return xerrors.Errorf("setup storage and api: %w", err)
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return xerrors.Errorf("create schema: %w", err)
		}

		processor := processor2.NewProcessor(rctx.db, rctx.api)
		if err := processor.InitHandler(ctx, cctx.Int("max-batch")); err != nil {
			return xerrors.Errorf("init processor: %w", err)
		}

		// Start the processor and wait for it to complete or to be cancelled.
		done := make(chan struct{})
		go func() {
			defer close(done)
			err = processor.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return nil
		case <-done:
			return err
		}

	},
}

var indexCmd = &cli.Command{
	Name:  "index",
	Usage: "Index the lotus blockchain",
	Action: func(cctx *cli.Context) error {
		err := setupLogging(cctx)
		if err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		if cctx.Bool("tracing") {
			jcfg, err := jaegerConfigFromCliContext(cctx)
			if err != nil {
				return xerrors.Errorf("read jeager config: %w", err)
			}
			closer, err := setupTracing(jcfg)
			if err != nil {
				return xerrors.Errorf("initialize tracing subsystem: %w", err)
			}
			defer closer()
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return err
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return xerrors.Errorf("create schema: %w", err)
		}

		indexer := indexer2.NewIndexer(rctx.db, rctx.api)
		if err := indexer.InitHandler(ctx); err != nil {
			return xerrors.Errorf("init indexer: %w", err)
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
		return nil, nil, xerrors.Errorf("get node api: %w", err)
	}

	db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"))
	if err != nil {
		return nil, nil, xerrors.Errorf("connect database: %w", err)
	}

	return ctx, &RunContext{api, closer, db}, nil
}

func setupTracing(cfg *jaegerConfig) (func(), error) {
	closer, err := jaeger.InstallNewPipeline(
		jaeger.WithAgentEndpoint(cfg.AgentEndpoint),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: cfg.ServiceName,
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: cfg.Sampler}),
	)
	if err != nil {
		return nil, xerrors.Errorf("install jaeger pipeline: %w", err)
	}

	return closer, nil
}

type jaegerConfig struct {
	ServiceName   string
	AgentEndpoint string
	Sampler       sdktrace.Sampler
}

func jaegerConfigFromCliContext(cctx *cli.Context) (*jaegerConfig, error) {
	cfg := jaegerConfig{
		ServiceName:   cctx.String("jaeger-service-name"),
		AgentEndpoint: fmt.Sprintf("%s:%d", cctx.String("jaeger-agent-host"), cctx.Int("jaeger-agent-port")),
	}

	switch cctx.String("jaeger-sampler-type") {
	case "probabilistic":
		cfg.Sampler = sdktrace.ProbabilitySampler(cctx.Float64("jaeger-sampler-param"))
	case "const":
		if cctx.Float64("jaeger-sampler-param") == 1 {
			cfg.Sampler = sdktrace.AlwaysSample()
		} else {
			cfg.Sampler = sdktrace.NeverSample()
		}
	default:
		return nil, fmt.Errorf("unsupported jaeger-sampler-type option: %s", cctx.String("jaeger-sampler-type"))
	}

	return &cfg, nil
}

func setupLogging(cctx *cli.Context) error {
	ll := cctx.String("log-level")
	if err := logging.SetLogLevel("*", ll); err != nil {
		return xerrors.Errorf("set log level: %w", err)
	}

	if err := logging.SetLogLevel("rpc", "error"); err != nil {
		return xerrors.Errorf("set rpc log level: %w", err)
	}

	llnamed := cctx.String("log-level-named")
	if llnamed == "" {
		return nil
	}

	for _, llname := range strings.Split(llnamed, ",") {
		parts := strings.Split(llname, ":")
		if len(parts) != 2 {
			return xerrors.Errorf("invalid named log level format: %q", llname)
		}
		if err := logging.SetLogLevel(parts[0], parts[1]); err != nil {
			return xerrors.Errorf("set named log level %q to %q: %w", parts[0], parts[1], err)
		}

	}

	return nil
}

func setupMetrics() error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := prom.NewGoCollector()
	procCollector := prom.NewProcessCollector(prom.ProcessCollectorOpts{})
	registry.MustRegister(goCollector, procCollector)
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "visor",
		Registry:  registry,
	})
	if err != nil {
		return err
	}

	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2 * time.Second)

	// register the metrics views of interest
	if err := view.Register(metrics.DefaultViews...); err != nil {
		return err
	}
	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		mux.Handle("/metrics", pe)
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		if err := http.ListenAndServe(":9991", mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
