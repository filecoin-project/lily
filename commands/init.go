package commands

import (
	"context"

	paramfetch "github.com/filecoin-project/go-paramfetch"
	lotusbuild "github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/config"
)

var initFlags struct {
	repo           string
	config         string
	importSnapshot string
}

var InitCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialise a lily repository.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "repo",
			Usage:       "Specify path where lily should store chain state.",
			EnvVars:     []string{"LILY_REPO"},
			Value:       "~/.lily",
			Destination: &initFlags.repo,
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "Specify path of config file to use.",
			EnvVars:     []string{"LILY_CONFIG"},
			Destination: &initFlags.config,
		},
		&cli.StringFlag{
			Name:        "import-snapshot",
			Usage:       "Import chain state from a given chain export file or url.",
			EnvVars:     []string{"LILY_SNAPSHOT"},
			Destination: &initFlags.importSnapshot,
		},
	},
	Action: func(c *cli.Context) error {
		lotuslog.SetupLogLevels()
		ctx := context.Background()
		{
			dir, err := homedir.Expand(initFlags.repo)
			if err != nil {
				log.Warnw("could not expand repo location", "error", err)
			} else {
				log.Infof("lily repo: %s", dir)
			}
		}

		r, err := repo.NewFS(initFlags.repo)
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		if initFlags.config != "" {
			if err := config.EnsureExists(initFlags.config); err != nil {
				return xerrors.Errorf("ensuring config is present at %q: %w", initFlags.config, err)
			}
			r.SetConfigPath(initFlags.config)
		}

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		if err := paramfetch.GetParams(lcli.ReqContext(c), lotusbuild.ParametersJSON(), lotusbuild.SrsJSON(), 0); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		if initFlags.importSnapshot != "" {
			if err := util.ImportChain(ctx, r, initFlags.importSnapshot, true); err != nil {
				return err
			}
		}

		return nil
	},
}
