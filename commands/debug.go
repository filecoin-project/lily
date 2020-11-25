package commands

import (
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
)

var Debug = &cli.Command{
	Name:  "debug",
	Usage: "Execute individual tasks without persisting them to the database",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor-head",
			Usage: "Process task in visor_processing_actors by head",
		},
	},
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		tcloser, err := setupTracing(cctx)
		if err != nil {
			return xerrors.Errorf("setup tracing: %w", err)
		}
		defer tcloser()

		ctx, rctx, err := SetupStorageAndAPI(cctx)
		if err != nil {
			return xerrors.Errorf("setup storage and api: %w", err)
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(ctx); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		p, err := actorstate.NewActorStateProcessor(rctx.db, rctx.opener, 0, 0, 0, 0, actorstate.SupportedActorCodes(), false)
		if err != nil {
			return err
		}

		return p.Debug(ctx, cctx.String("actor-head"), os.Stdout)
	},
}
