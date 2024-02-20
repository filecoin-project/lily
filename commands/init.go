package commands

import (
	"context"
	"fmt"

	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/config"

	lotusbuild "github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/node/repo"
)

var initFlags struct {
	repo                   string
	config                 string
	importSnapshot         string
	backfillTipsetKeyRange int
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
		&cli.IntFlag{
			Name:        "backfill-tipsetkey-range",
			Usage:       "Determine the extent of backfilling from the head.",
			EnvVars:     []string{"LILY_BACKFILL_TIPSETKEY_RANGE"},
			Value:       3600,
			Destination: &initFlags.backfillTipsetKeyRange,
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
				log.Infof("lotus repo: %s", dir)
			}
		}

		r, err := repo.NewFS(initFlags.repo)
		if err != nil {
			return fmt.Errorf("opening fs repo: %w", err)
		}

		if initFlags.config != "" {
			if err := config.EnsureExists(initFlags.config); err != nil {
				return fmt.Errorf("ensuring config is present at %q: %w", initFlags.config, err)
			}
			r.SetConfigPath(initFlags.config)
		}

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return fmt.Errorf("repo init error: %w", err)
		}

		if err := paramfetch.GetParams(lcli.ReqContext(c), lotusbuild.ParametersJSON(), lotusbuild.SrsJSON(), 0); err != nil {
			return fmt.Errorf("fetching proof parameters: %w", err)
		}

		if initFlags.importSnapshot != "" {
			if err := util.ImportChain(ctx, r, initFlags.importSnapshot, true, initFlags.backfillTipsetKeyRange); err != nil {
				return err
			}
		}

		return nil
	},
}
