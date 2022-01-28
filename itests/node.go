package itests

import (
	"context"
	"github.com/filecoin-project/lily/commands"
	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/lens/lily/modules"
	lutil "github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/lib/peermgr"
	"github.com/filecoin-project/lotus/node"
	lotusmodules "github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"io/fs"
	"io/ioutil"
	"testing"
)

type TestNodeConfig struct {
	LilyConfig  *config.Conf
	CacheConfig *lutil.CacheConfig
	RepoPath    string
	Snapshot    fs.File
	Genesis     fs.File
	ApiEndpoint string
}

func NewTestNode(t testing.TB, ctx context.Context, cfg *TestNodeConfig) (lily.LilyAPI, node.StopFunc) {
	r, err := repo.NewFS(cfg.RepoPath)
	require.NoError(t, err)

	err = r.Init(repo.FullNode)
	require.NoError(t, err)

	err = util.ImportFromFsFile(ctx, r, cfg.Snapshot, true)
	require.NoError(t, err)

	genBytes, err := ioutil.ReadAll(cfg.Genesis)
	require.NoError(t, err)

	genesis := node.Options()
	if len(genBytes) > 0 {
		genesis = node.Override(new(lotusmodules.Genesis), lotusmodules.LoadGenesis(genBytes))
	}
	liteModeDeps := node.Options()
	shutdown := make(chan struct{})
	var api lily.LilyAPI
	stop, err := node.New(ctx,
		// Start Sentinel Dep injection
		commands.LilyNodeAPIOption(&api),
		node.Override(new(*config.Conf), func() (*config.Conf, error) {
			return cfg.LilyConfig, nil
		}),
		node.Override(new(*lutil.CacheConfig), func() (*lutil.CacheConfig, error) {
			return cfg.CacheConfig, nil
		}),
		node.Override(new(*events.Events), modules.NewEvents),
		node.Override(new(*schedule.Scheduler), schedule.NewSchedulerDaemon),
		node.Override(new(*storage.Catalog), modules.NewStorageCatalog),
		// End Injection

		node.Override(new(dtypes.Bootstrapper), false),
		node.Override(new(dtypes.ShutdownChan), shutdown),
		node.Base(),
		node.Repo(r),

		node.Override(new(dtypes.UniversalBlockstore), modules.NewCachingUniversalBlockstore),

		// Inject a custom StateManager, must be done after the node.Online() call as we are
		// overriding the OG lotus StateManager.
		node.Override(new(*stmgr.StateManager), modules.StateManager),
		node.Override(new(stmgr.ExecMonitor), modules.NewBufferedExecMonitor),
		// End custom StateManager injection.
		genesis,
		liteModeDeps,
		// run the node in offline mode
		node.Unset(node.RunPeerMgrKey),
		node.Unset(new(*peermgr.PeerMgr)),

		node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
			apima, err := multiaddr.NewMultiaddr(cfg.ApiEndpoint)
			if err != nil {
				return err
			}
			return lr.SetAPIEndpoint(apima)
		}),
	)
	require.NoError(t, err)
	return api, stop
}
