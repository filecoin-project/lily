package main

import (
	"context"
	_ "net/http/pprof"
	"os"

	_ "github.com/lib/pq"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	lens "github.com/filecoin-project/sentinel-visor/lens/lotus"
	vapi "github.com/filecoin-project/sentinel-visor/lens/lotus"
	indexer2 "github.com/filecoin-project/sentinel-visor/services/indexer"
	processor2 "github.com/filecoin-project/sentinel-visor/services/processor"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start visor",
	Subcommands: []*cli.Command{
		processorCmd,
		indexerCmd,
	},
}

var processorCmd = &cli.Command{
	Name:  "processor",
	Usage: "Non-singleton processor of the lotus blockchain",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 50,
		},
	},
	Action: func(cctx *cli.Context) error {
		ll := cctx.String("log-level")
		if err := logging.SetLogLevel("*", ll); err != nil {
			return err
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return err
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("closing base", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return err
		}

		processor := processor2.NewProcessor(rctx.db, rctx.api)
		if err := processor.InitHandler(ctx, cctx.Int("max-batch")); err != nil {
			return err
		}
		processor.Start(ctx)

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		<-ctx.Done()
		os.Exit(0)
		return nil
	},
}

var indexerCmd = &cli.Command{
	Name:  "indexer",
	Usage: "Singleton indexer of the lotus blockchain",
	Action: func(cctx *cli.Context) error {
		ll := cctx.String("log-level")
		if err := logging.SetLogLevel("*", ll); err != nil {
			return err
		}

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return err
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(); err != nil {
				log.Errorw("closing statebase", "error", err)
			}
		}()

		if err := rctx.db.CreateSchema(); err != nil {
			return err
		}

		indexer := indexer2.NewIndexer(rctx.db, rctx.api)
		if err := indexer.InitHandler(ctx); err != nil {
			return err
		}

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		// Start the indexer and wait for it to complete or to be cancelled.
		done := make(chan struct{})
		go func() {
			defer close(done)
			// TODO if the lotus daemon hangs up Start will exit. It should restart and wait for lotus to come back online.
			err = indexer.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return nil
		case <-done:
			return err
		}
	},
}

type RunContext struct {
	api    lens.API
	closer lens.APICloser
	db     *storage.Database
}

func setupStorageAndAPI(cctx *cli.Context) (context.Context, *RunContext, error) {
	ctx, api, closer, err := vapi.GetFullNodeAPI(cctx)
	if err != nil {
		return nil, nil, err
	}

	db, err := storage.NewDatabase(ctx, cctx.String("db"))
	if err != nil {
		return nil, nil, err
	}

	return ctx, &RunContext{api, closer, db}, nil
}
