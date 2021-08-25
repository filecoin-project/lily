package commands

import (
	"context"
	"io/ioutil"

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
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/commands/util"
	"github.com/filecoin-project/sentinel-visor/config"
	"github.com/filecoin-project/sentinel-visor/lens/lily"
	"github.com/filecoin-project/sentinel-visor/lens/lily/modules"
	"github.com/filecoin-project/sentinel-visor/schedule"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var clientAPIFlags struct {
	apiAddr  string
	apiToken string
}

var clientAPIFlag = &cli.StringFlag{
	Name:        "api",
	Usage:       "Address of visor api in multiaddr format.",
	EnvVars:     []string{"VISOR_API"},
	Value:       "/ip4/127.0.0.1/tcp/1234",
	Destination: &clientAPIFlags.apiAddr,
}

var clientTokenFlag = &cli.StringFlag{
	Name:        "api-token",
	Usage:       "Authentication token for visor api.",
	EnvVars:     []string{"VISOR_API_TOKEN"},
	Value:       "",
	Destination: &clientAPIFlags.apiToken,
}

// clientAPIFlagSet are used by commands that act as clients of a daemon's API
var clientAPIFlagSet = []cli.Flag{
	clientAPIFlag,
	clientTokenFlag,
}

type daemonOpts struct {
	repo          string
	bootstrap     bool
	config        string
	commentConfig bool
	genesis       string
}

var daemonFlags daemonOpts

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start a visor daemon process.",
	Description: `Starts visor in daemon mode with its own blockstore.
In daemon mode visor synchronizes with the filecoin network and runs jobs such
as walk and watch against its local blockstore. This gives better performance
than operating against an external blockstore but requires more disk space and
memory.

Before starting the daemon for the first time the blockstore must be initialized
and synchronized. Visor can use a copy of an already synchronized Lotus node
repository or can initialize its own empty one and import a snapshot to catch
up to the chain.

To initialize an empty visor blockstore repository and import an initial
snapshot of the chain:

  visor init --repo=<path> --import-snapshot=<url>

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

  visor daemon --repo=<path> --config=<path>/config.toml

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
			Usage:       "Specify path where visor should store chain state.",
			EnvVars:     []string{"VISOR_REPO"},
			Value:       "~/.lotus",
			Destination: &daemonFlags.repo,
		},
		&cli.BoolFlag{
			Name:        "bootstrap",
			Usage:       "If set to false the daemon will not connected to peers or sync the chain.",
			EnvVars:     []string{"VISOR_BOOTSTRAP"},
			Value:       true,
			Destination: &daemonFlags.bootstrap,
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Specify path of config file to use.",
			EnvVars:     []string{"VISOR_CONFIG"},
			Value:       "~/.lotus/config.toml",
			Destination: &daemonFlags.config,
		},
		&cli.BoolFlag{
			Name:        "default-config",
			Usage:       "If true the daemon config file will be overwritten with default configuration values.",
			Value:       false,
			Destination: &daemonFlags.commentConfig,
		},
		&cli.StringFlag{
			Name:        "genesis",
			Usage:       "Genesis file to use for first node run.",
			EnvVars:     []string{"VISOR_GENESIS"},
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

		tcloser, err := setupTracing(VisorTracingFlags)
		if err != nil {
			return xerrors.Errorf("setup tracing: %w", err)
		}
		defer tcloser()

		ctx := context.Background()

		repoPath, err := homedir.Expand(daemonFlags.repo)
		if err != nil {
			return xerrors.Errorf("could not expand repo location: %w", err)
		}
		log.Infof("repo path: %s", repoPath)
		configPath, err := homedir.Expand(daemonFlags.config)
		if err != nil {
			return xerrors.Errorf("could not expand config location: %w", err)
		}
		log.Infof("config path: %s", configPath)

		r, err := repo.NewFS(repoPath)
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		if err := config.EnsureExists(configPath, daemonFlags.commentConfig); err != nil {
			return xerrors.Errorf("ensuring config is present at %q: %w", configPath, err)
		}
		r.SetConfigPath(configPath)

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
			node.Override(new(*config.Conf), modules.LoadConf(configPath)),
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
					apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" +
						c.String("api"))
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
