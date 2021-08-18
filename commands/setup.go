package commands

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	lotusmetrics "github.com/filecoin-project/lotus/metrics"
	logging "github.com/ipfs/go-log/v2"
	metricsprom "github.com/ipfs/go-metrics-prometheus"
	_ "github.com/lib/pq"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/xerrors"

	lens "github.com/filecoin-project/sentinel-visor/lens"
	carapi "github.com/filecoin-project/sentinel-visor/lens/carrepo"
	vapi "github.com/filecoin-project/sentinel-visor/lens/lotus"
	repoapi "github.com/filecoin-project/sentinel-visor/lens/lotusrepo"
	sqlapi "github.com/filecoin-project/sentinel-visor/lens/sqlrepo"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/version"
)

var log = logging.Logger("visor/commands")

func setupDatabase(cctx *cli.Context) (*storage.Database, error) {
	ctx := cctx.Context
	db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"), cctx.String("name"), cctx.String("schema"), cctx.Bool("db-allow-upsert"))
	if err != nil {
		return nil, xerrors.Errorf("new database: %w", err)
	}

	if err := db.Connect(ctx); err != nil {
		if !errors.Is(err, storage.ErrSchemaTooOld) || !cctx.Bool("allow-schema-migration") {
			return nil, xerrors.Errorf("connect database: %w", err)
		}

		log.Infof("connect database: %v", err.Error())

		// Schema is out of data and we're allowed to do schema migrations
		log.Info("Migrating schema to latest version")
		err := db.MigrateSchema(ctx)
		if err != nil {
			return nil, xerrors.Errorf("migrate schema: %w", err)
		}

		// Try to connect again
		if err := db.Connect(ctx); err != nil {
			return nil, xerrors.Errorf("connect database: %w", err)
		}
	}

	// Make sure the schema is a compatible with what this version of Visor requires
	if err := db.VerifyCurrentSchema(ctx); err != nil {
		db.Close(ctx) // nolint: errcheck
		return nil, xerrors.Errorf("verify schema: %w", err)
	}

	return db, nil
}

func setupLens(cctx *cli.Context) (lens.APIOpener, lens.APICloser, error) {
	switch cctx.String("lens") {
	case "lotus":
		return vapi.NewAPIOpener(cctx, 100_000)
	case "lotusrepo":
		return repoapi.NewAPIOpener(cctx)
	case "carrepo":
		return carapi.NewAPIOpener(cctx)
	case "sql":
		return sqlapi.NewAPIOpener(cctx)
	default:
		return nil, nil, xerrors.Errorf("unsupported lens type: %s", cctx.String("lens"))
	}
}

func setupTracing(cctx *cli.Context) (func(), error) {
	if !cctx.Bool("tracing") {
		global.SetTracerProvider(trace.NoopTracerProvider())
	}

	jcfg, err := jaegerConfigFromCliContext(cctx)
	if err != nil {
		return nil, xerrors.Errorf("read jeager config: %w", err)
	}

	closer, err := jaeger.InstallNewPipeline(
		jaeger.WithAgentEndpoint(jcfg.AgentEndpoint),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: jcfg.ServiceName,
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: jcfg.Sampler}),
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
		cfg.Sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cctx.Float64("jaeger-sampler-param")))
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
	//ll := cctx.String("log-level")
	if err := logging.SetLogLevel("*", "error"); err != nil {
		return xerrors.Errorf("set log level: %w", err)
	}

	if err := logging.SetLogLevelRegex("visor/*", "debug"); err != nil {
		panic(err)
	}

	llnamed := cctx.String("log-level-named")
	if llnamed != "" {
		for _, llname := range strings.Split(llnamed, ",") {
			parts := strings.Split(llname, ":")
			if len(parts) != 2 {
				return xerrors.Errorf("invalid named log level format: %q", llname)
			}
			if err := logging.SetLogLevel(parts[0], parts[1]); err != nil {
				return xerrors.Errorf("set named log level %q to %q: %w", parts[0], parts[1], err)
			}

		}
	}

	log.Infof("Visor version:%s", version.String())
	if err := logging.SetLogLevelRegex("visor/*", "info"); err != nil {
		panic(err)
	}

	return nil
}

func setupMetrics(cctx *cli.Context) error {
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

	views := []*view.View{}
	views = append(views, metrics.DefaultViews...)        // visor metrics
	views = append(views, lotusmetrics.ChainNodeViews...) // lotus chain metrics

	// register the metrics views of interest
	if err := view.Register(views...); err != nil {
		return err
	}

	// some libraries like ipfs/go-ds-measure and ipfs/go-ipfs-blockstore
	// use ipfs/go-metrics-interface. This injects a Prometheus exporter
	// for those. Metrics are exported to the default registry.
	if err := metricsprom.Inject(); err != nil {
		log.Warnf("unable to inject prometheus ipfs/go-metrics exporter; some metrics will be unavailable; err: %s", err)
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
		if err := http.ListenAndServe(cctx.String("prometheus-port"), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
