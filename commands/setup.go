package commands

import (
	"context"
	"errors"
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
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/xerrors"

	lens "github.com/filecoin-project/sentinel-visor/lens"
	carapi "github.com/filecoin-project/sentinel-visor/lens/carrepo"
	vapi "github.com/filecoin-project/sentinel-visor/lens/lotus"
	repoapi "github.com/filecoin-project/sentinel-visor/lens/lotusrepo"
	s3api "github.com/filecoin-project/sentinel-visor/lens/s3repo"
	sqlapi "github.com/filecoin-project/sentinel-visor/lens/sqlrepo"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var log = logging.Logger("visor")

type RunContext struct {
	opener lens.APIOpener
	closer lens.APICloser
	db     *storage.Database
}

func setupStorageAndAPI(cctx *cli.Context) (context.Context, *RunContext, error) {
	var opener lens.APIOpener // the api opener that is used by tasks
	var closer lens.APICloser // a closer that cleans up the opener when exiting the application
	var err error

	ctx := cctx.Context

	if cctx.String("lens") == "lotus" {
		opener, closer, err = vapi.NewAPIOpener(cctx, 10_000)
	} else if cctx.String("lens") == "lotusrepo" {
		opener, closer, err = repoapi.NewAPIOpener(cctx)
	} else if cctx.String("lens") == "carrepo" {
		opener, closer, err = carapi.NewAPIOpener(cctx)
	} else if cctx.String("lens") == "sql" {
		opener, closer, err = sqlapi.NewAPIOpener(cctx)
	} else if cctx.String("lens") == "s3" {
		opener, closer, err = s3api.NewAPIOpener(cctx)
	}
	if err != nil {
		return nil, nil, xerrors.Errorf("get node api: %w", err)
	}

	db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"))
	if err != nil {
		closer()
		return nil, nil, xerrors.Errorf("new database: %w", err)
	}

	if err := db.Connect(ctx); err != nil {
		if !errors.Is(err, storage.ErrSchemaTooOld) || !cctx.Bool("allow-schema-migration") {
			return nil, nil, xerrors.Errorf("connect database: %w", err)
		}

		log.Infof("connect database: %v", err.Error())

		// Schema is out of data and we're allowed to do schema migrations
		log.Info("Migrating schema to latest version")
		err := db.MigrateSchema(ctx)
		if err != nil {
			closer()
			return nil, nil, xerrors.Errorf("migrate schema: %w", err)
		}

		// Try to connect again
		if err := db.Connect(ctx); err != nil {
			closer()
			return nil, nil, xerrors.Errorf("connect database: %w", err)
		}
	}

	// Make sure the schema is a compatible with what this version of Visor requires
	if err := db.VerifyCurrentSchema(ctx); err != nil {
		closer()
		db.Close(ctx)
		return nil, nil, xerrors.Errorf("verify schema: %w", err)
	}

	return ctx, &RunContext{
		opener: opener,
		closer: closer,
		db:     db,
	}, nil
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

const (
	ChainHeadIndexerLockID         = 98981111
	ChainHistoryIndexerLockID      = 98981112
	ChainVisRefresherLockID        = 98981113
	ProcessingStatsRefresherLockID = 98981114
)

func NewGlobalSingleton(id int64, d *storage.Database) *GlobalSingleton {
	return &GlobalSingleton{
		LockID:  storage.AdvisoryLock(id),
		Storage: d,
	}
}

// GlobalSingleton is a task locker that ensures only one task can run across all processes
type GlobalSingleton struct {
	LockID  storage.AdvisoryLock
	Storage *storage.Database
}

func (g *GlobalSingleton) Lock(ctx context.Context) error {
	return g.LockID.LockExclusive(ctx, g.Storage.DB)
}

func (g *GlobalSingleton) Unlock(ctx context.Context) error {
	return g.LockID.UnlockExclusive(ctx, g.Storage.DB)
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
		if err := http.ListenAndServe(cctx.String("prometheus-port"), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
