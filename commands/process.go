package commands

import (
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	processor2 "github.com/filecoin-project/sentinel-visor/services/processor"
)

// TODO: rework to use new scheduler with just actor state and message tasks
var Process = &cli.Command{
	Name:  "process",
	Usage: "Process indexed blocks of the lotus blockchain",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 10,
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

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return xerrors.Errorf("setup storage and api: %w", err)
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(ctx); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		processor := processor2.NewProcessor(rctx.db, rctx.api)
		if err := processor.InitHandler(ctx, cctx.Int("max-batch")); err != nil {
			return xerrors.Errorf("init processor: %w", err)
		}

		// Start the processor and wait for it to complete or to be cancelled.
		done := make(chan struct{})
		go func() {
			defer close(done)
			err = processor.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return nil
		case <-done:
			return err
		}

	},
}
