package commands

import (
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	octrace "go.opencensus.io/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/bridge/opencensus"

	"contrib.go.opencensus.io/exporter/prometheus"
	lotusmetrics "github.com/filecoin-project/lotus/metrics"
	asynqmetrics "github.com/hibiken/asynq/x/metrics"
	logging "github.com/ipfs/go-log/v2"
	metricsprom "github.com/ipfs/go-metrics-prometheus"
	_ "github.com/lib/pq"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/version"
)

var log = logging.Logger("lily/commands")

type VisorLogOpts struct {
	LogLevel      string
	LogLevelNamed string
}

var VisorLogFlags VisorLogOpts

type VisorTracingOpts struct {
	Enabled            bool
	ServiceName        string
	ProviderURL        string
	JaegerSamplerParam float64
}

var VisorTracingFlags VisorTracingOpts

type VisorMetricOpts struct {
	PrometheusPort string
	RedisNetwork   string
	RedisAddr      string
	RedisUsername  string
	RedisPassword  string
	RedisDB        int
}

var VisorMetricFlags VisorMetricOpts

func setupLogging(flags VisorLogOpts) error {
	ll := flags.LogLevel
	if err := logging.SetLogLevel("*", ll); err != nil {
		return xerrors.Errorf("set log level: %w", err)
	}

	llnamed := flags.LogLevelNamed
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

	return nil
}

func setupMetrics(flags VisorMetricOpts) error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := collectors.NewGoCollector()
	procCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "visor",
		Registry:  registry,
	})
	if err != nil {
		return err
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     flags.RedisAddr,
		DB:       flags.RedisDB,
		Password: flags.RedisPassword,
		Username: flags.RedisUsername,
	})
	registry.MustRegister(
		goCollector,
		procCollector,
		asynqmetrics.NewQueueMetricsCollector(inspector),
	)
	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2 * time.Second)

	views := []*view.View{}
	views = append(views, metrics.DefaultViews...)        // lily metrics
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
		if err := http.ListenAndServe(flags.PrometheusPort, mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}

func setupTracing(flags VisorTracingOpts) error {
	if !flags.Enabled {
		return nil
	}

	tp, err := metrics.NewJaegerTraceProvider(VisorTracingFlags.ServiceName, VisorTracingFlags.ProviderURL, VisorTracingFlags.JaegerSamplerParam)
	if err != nil {
		return xerrors.Errorf("setup tracing: %w", err)
	}
	otel.SetTracerProvider(tp)
	// upgrades libraries (lotus) that use OpenCensus to OpenTelemetry to facilitate a migration.
	tracer := tp.Tracer(VisorTracingFlags.ServiceName)
	octrace.DefaultTracer = opencensus.NewTracer(tracer)

	return nil
}
