package commands

import (
	"context"
	"io/ioutil"
	"path/filepath"

	paramfetch "github.com/filecoin-project/go-paramfetch"
	lotusbuild "github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/stmgr"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/peermgr"
	"github.com/filecoin-project/lotus/node"
	lotusmodules "github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/lens/lily/modules"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/lily/storage"
)

var clientAPIFlags struct {
	apiAddr  string
	apiToken string
}

var clientAPIFlag = &cli.StringFlag{
	Name:        "api",
	Usage:       "Address of lily api in multiaddr format.",
	EnvVars:     []string{"LILY_API"},
	Value:       "/ip4/127.0.0.1/tcp/1234",
	Destination: &clientAPIFlags.apiAddr,
}

var clientTokenFlag = &cli.StringFlag{
	Name:        "api-token",
	Usage:       "Authentication token for lily api.",
	EnvVars:     []string{"LILY_API_TOKEN"},
	Value:       "",
	Destination: &clientAPIFlags.apiToken,
}

// clientAPIFlagSet are used by commands that act as clients of a daemon's API
var clientAPIFlagSet = []cli.Flag{
	clientAPIFlag,
	clientTokenFlag,
}

type daemonOpts struct {
	repo      string
	bootstrap bool // TODO: is this necessary - do we want to run lily in this mode?
	config    string
	genesis   string
}

var daemonFlags daemonOpts

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start a lily daemon process.",
	Description: `Starts lily in daemon mode with its own blockstore.
In daemon mode lily synchronizes with the filecoin network and runs jobs such
as walk and watch against its local blockstore. This gives better performance
than operating against an external blockstore but requires more disk space and
memory.

Before starting the daemon for the first time the blockstore must be initialized
and synchronized. Visor can use a copy of an already synchronized Lotus node
repository or can initialize its own empty one and import a snapshot to catch
up to the chain.

To initialize an empty lily blockstore repository and import an initial
snapshot of the chain:

  lily init --repo=<path> --import-snapshot=<url>

This will initialize a blockstore repository at <path> and import chain state
from the snapshot at <url>. The type of snapshot needed depends on the type
of jobs expected to be run by the daemon.

Watch jobs only require current actor state to be imported since incoming
tipsets will be used to keep actor states up to date. The lightweight and full
chain snapshots available at https://docs.filecoin.io/get-started/lotus/chain/
are suitable to initialize the daemon for watch jobs.

Historic walks will require full actor states for the range of heights covered
by the walk. These may be created from an existing Lotus node using the
export command, provided receipts are also included in the export. See the
Lotus documentation for full details.

Once the repository is initialized, start the daemon:

  lily daemon --repo=<path> --config=<path>/config.toml

Visor will connect to the filecoin network and begin synchronizing with the
chain. To check the synchronization status use 'visor sync status' or
'visor sync wait'.

Jobs may be started on the daemon at any time. A watch job will wait for the
daemon to become synchronized before extracting data and will pause if the
daemon falls out of sync. Start a watch using 'visor watch'.

A walk job will start immediately. Start a walk using 'visor walk'. A walk may
only be performed between heights that have been synchronized with the network.

Note that jobs are not persisted between restarts of the daemon. See
'visor help job' for more information on managing jobs being run by the daemon.
`,

	Flags: []cli.Flag{
		clientAPIFlag,
		&cli.StringFlag{
			Name:        "repo",
			Usage:       "Specify path where lily should store chain state.",
			EnvVars:     []string{"LILY_REPO"},
			Value:       "~/.lily",
			Destination: &daemonFlags.repo,
		},
		&cli.BoolFlag{
			Name: "bootstrap",
			// TODO: usage description
			EnvVars:     []string{"LILY_BOOTSTRAP"},
			Value:       true,
			Destination: &daemonFlags.bootstrap,
			Hidden:      true, // hide until we decide if we want to keep this.
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Specify path of config file to use.",
			EnvVars:     []string{"LILY_CONFIG"},
			Destination: &daemonFlags.config,
		},
		&cli.StringFlag{
			Name:        "genesis",
			Usage:       "Genesis file to use for first node run.",
			EnvVars:     []string{"LILY_GENESIS"},
			Destination: &daemonFlags.genesis,
		},
	},
	Action: func(c *cli.Context) error {
		lotuslog.SetupLogLevels()

		if err := setupLogging(VisorLogFlags); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		if err := setupMetrics(VisorMetricFlags); err != nil {
			return xerrors.Errorf("setup metrics: %w", err)
		}

		if VisorTracingFlags.Tracing {
			tp, err := NewJaegerTraceProvider(VisorTracingFlags)
			if err != nil {
				return xerrors.Errorf("setup tracing: %w", err)
			}
			otel.SetTracerProvider(tp)
		} else {
			otel.SetTracerProvider(trace.NewNoopTracerProvider())
		}

		ctx := context.Background()
		repoDir, err := homedir.Expand(daemonFlags.repo)
		if err != nil {
			log.Warnw("could not expand repo location", "error", err)
		} else {
			log.Infof("visor repo: %s", repoDir)
		}

		r, err := repo.NewFS(daemonFlags.repo)
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		if daemonFlags.config == "" {
			daemonFlags.config = filepath.Join(repoDir, "config.toml")
		} else {
			daemonFlags.config, err = homedir.Expand(daemonFlags.config)
			if err != nil {
				log.Warnw("could not expand repo location", "error", err)
			} else {
				log.Infof("visor config: %s", repoDir)
			}
		}

		if err := config.EnsureExists(daemonFlags.config); err != nil {
			return xerrors.Errorf("ensuring config is present at %q: %w", daemonFlags.config, err)
		}
		r.SetConfigPath(daemonFlags.config)

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		if err := paramfetch.GetParams(lcli.ReqContext(c), lotusbuild.ParametersJSON(), lotusbuild.SrsJSON(), 0); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		var genBytes []byte
		if c.String("genesis") != "" {
			genBytes, err = ioutil.ReadFile(daemonFlags.genesis)
			if err != nil {
				return xerrors.Errorf("reading genesis: %w", err)
			}
		} else {
			genBytes = lotusbuild.MaybeGenesis()
		}

		genesis := node.Options()
		if len(genBytes) > 0 {
			genesis = node.Override(new(lotusmodules.Genesis), lotusmodules.LoadGenesis(genBytes))
		}

		isBootstrapper := false
		shutdown := make(chan struct{})
		liteModeDeps := node.Options()
		var api lily.LilyAPI
		stop, err := node.New(ctx,
			// Start Sentinel Dep injection
			LilyNodeAPIOption(&api),
			node.Override(new(*config.Conf), modules.LoadConf(daemonFlags.config)),
			node.Override(new(*events.Events), modules.NewEvents),
			node.Override(new(*schedule.Scheduler), schedule.NewSchedulerDaemon),
			node.Override(new(*storage.Catalog), modules.NewStorageCatalog),
			// End Injection

			node.Override(new(dtypes.Bootstrapper), isBootstrapper),
			node.Override(new(dtypes.ShutdownChan), shutdown),
			node.Online(),
			node.Repo(r),

			// Inject a custom StateManager, must be done after the node.Online() call as we are
			// overriding the OG lotus StateManager.
			node.Override(new(*stmgr.StateManager), modules.StateManager),
			node.Override(new(stmgr.ExecMonitor), modules.NewBufferedExecMonitor),
			// End custom StateManager injection.
			genesis,
			liteModeDeps,

			node.ApplyIf(func(s *node.Settings) bool { return c.IsSet("api") },
				node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
					apima, err := multiaddr.NewMultiaddr(clientAPIFlags.apiAddr)
					if err != nil {
						return err
					}
					return lr.SetAPIEndpoint(apima)
				})),
			node.ApplyIf(func(s *node.Settings) bool { return !daemonFlags.bootstrap },
				node.Unset(node.RunPeerMgrKey),
				node.Unset(new(*peermgr.PeerMgr)),
			),
		)
		if err != nil {
			return xerrors.Errorf("initializing node: %w", err)
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return xerrors.Errorf("getting api endpoint: %w", err)
		}

		// TODO: properly parse api endpoint (or make it a URL)
		maxAPIRequestSize := int64(0)
		return util.ServeRPC(api, stop, endpoint, shutdown, maxAPIRequestSize)
	},
}
