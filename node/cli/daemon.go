package cli

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/vm"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/lib/peermgr"
	"github.com/filecoin-project/lotus/node"
	lotusmodules "github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"gopkg.in/cheggaaa/pb.v1"

	sentinelAPI "github.com/filecoin-project/sentinel-visor/node/api"
	"github.com/filecoin-project/sentinel-visor/node/impl"
)

var log = logging.Logger("lotus-visor")

var DaemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start a lily daemon process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "api",
			Value: "1234",
		},
		&cli.BoolFlag{
			Name:  "bootstrap",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "config",
			Usage: "specify path of config file to use",
		},
		&cli.StringFlag{
			Name:  "import-snapshot",
			Usage: "import chain state from a given chain export file or url",
		},
		&cli.StringFlag{
			Name:  "genesis",
			Usage: "genesis file to use for first node run",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		{
			dir, err := homedir.Expand(cctx.String("repo"))
			if err != nil {
				log.Warnw("could not expand repo location", "error", err)
			} else {
				log.Infof("lotus repo: %s", dir)
			}
		}

		r, err := repo.NewFS(cctx.String("repo"))
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		if cctx.String("config") != "" {
			r.SetConfigPath(cctx.String("config"))
		}

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		if err := paramfetch.GetParams(lcli.ReqContext(cctx), build.ParametersJSON(), 0); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		var genBytes []byte
		if cctx.String("genesis") != "" {
			genBytes, err = ioutil.ReadFile(cctx.String("genesis"))
			if err != nil {
				return xerrors.Errorf("reading genesis: %w", err)
			}
		} else {
			genBytes = build.MaybeGenesis()
		}

		snapshot := cctx.String("import-snapshot")
		if snapshot != "" {
			if err := ImportChain(ctx, r, snapshot, true); err != nil {
				return err
			}
		}

		genesis := node.Options()
		if len(genBytes) > 0 {
			genesis = node.Override(new(lotusmodules.Genesis), lotusmodules.LoadGenesis(genBytes))
		}

		isBootstrapper := false
		shutdownChan := make(chan struct{})
		liteModeDeps := node.Options()
		var api sentinelAPI.LilyNode
		stop, err := node.New(ctx,
			// Start Sentinel Dep injection
			LilyNodeAPIOption(&api),
			node.Override(new(*events.Events), impl.NewEvents),
			// End Injection

			node.Override(new(dtypes.Bootstrapper), isBootstrapper),
			node.Override(new(dtypes.ShutdownChan), shutdownChan),
			node.Online(),
			node.Repo(r),

			genesis,
			liteModeDeps,

			node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("api") },
				node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
					apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" +
						cctx.String("api"))
					if err != nil {
						return err
					}
					return lr.SetAPIEndpoint(apima)
				})),
			node.ApplyIf(func(s *node.Settings) bool { return !cctx.Bool("bootstrap") },
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
		return serveRPC(api, stop, endpoint, shutdownChan, int64(cctx.Int("api-max-req-size")))
	},
}

// Lily Node settings for injection into lotus node.
func LilyNodeAPIOption(out *sentinelAPI.LilyNode, fopts ...node.Option) node.Option {
	resAPI := &impl.LilyNodeAPI{}
	opts := node.Options(
		node.WithRepoType(repo.FullNode),
		node.Options(fopts...),
		node.WithInvokesKey(node.ExtractApiKey, resAPI),
	)
	*out = resAPI
	return opts
}

func ImportChain(ctx context.Context, r repo.Repo, fname string, snapshot bool) (err error) {
	var rd io.Reader
	var l int64
	if strings.HasPrefix(fname, "http://") || strings.HasPrefix(fname, "https://") {
		resp, err := http.Get(fname) //nolint:gosec
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("non-200 response: %d", resp.StatusCode)
		}

		rd = resp.Body
		l = resp.ContentLength
	} else {
		fname, err = homedir.Expand(fname)
		if err != nil {
			return err
		}

		fi, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer fi.Close() //nolint:errcheck

		st, err := os.Stat(fname)
		if err != nil {
			return err
		}

		rd = fi
		l = st.Size()
	}

	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	bs, err := lr.Blockstore(ctx, repo.BlockstoreChain)
	if err != nil {
		return xerrors.Errorf("failed to open blockstore: %w", err)
	}

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	j, err := journal.OpenFSJournal(lr, journal.EnvDisabledEvents())
	if err != nil {
		return xerrors.Errorf("failed to open journal: %w", err)
	}

	cst := store.NewChainStore(bs, bs, mds, vm.Syscalls(ffiwrapper.ProofVerifier), j)
	defer cst.Close() //nolint:errcheck

	log.Infof("importing chain from %s...", fname)

	bufr := bufio.NewReaderSize(rd, 1<<20)

	bar := pb.New64(l)
	br := bar.NewProxyReader(bufr)
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.ShowSpeed = true
	bar.Units = pb.U_BYTES

	bar.Start()
	ts, err := cst.Import(br)
	bar.Finish()

	if err != nil {
		return xerrors.Errorf("importing chain failed: %w", err)
	}

	if err := cst.FlushValidationCache(); err != nil {
		return xerrors.Errorf("flushing validation cache failed: %w", err)
	}

	gb, err := cst.GetTipsetByHeight(ctx, 0, ts, true)
	if err != nil {
		return err
	}

	err = cst.SetGenesis(gb.Blocks()[0])
	if err != nil {
		return err
	}

	stm := stmgr.NewStateManager(cst)

	if !snapshot {
		log.Infof("validating imported chain...")
		if err := stm.ValidateChain(ctx, ts); err != nil {
			return xerrors.Errorf("chain validation failed: %w", err)
		}
	}

	log.Infof("accepting %s as new head", ts.Cids())
	if err := cst.ForceHeadSilent(ctx, ts); err != nil {
		return err
	}

	return nil
}
