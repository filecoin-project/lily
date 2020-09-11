package main

import (
	//"github.com/frrist/visor/model"
	"github.com/filecoin-project/visor/services"
	"github.com/filecoin-project/visor/storage"
	_ "net/http/pprof"
	"os"

	_ "github.com/lib/pq"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	vapi "github.com/filecoin-project/visor/lens/lotus"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start visor",
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

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		/*
			traceFn := metrics.InitTracer()
			defer traceFn()
		*/

		ctx, api, closer, err := vapi.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		db, err := storage.NewDatabase(ctx, cctx.String("db"))
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.CreateSchema(); err != nil {
			return err
		}

		publisher := services.NewPublisher(db)
		scheduler := services.NewScheduler(api, publisher)
		indexer := services.NewIndexer(db, api)
		processor := services.NewProcessor(db, indexer, scheduler, api)

		if err := indexer.InitHandler(ctx); err != nil {
			return err
		}

		if err := processor.InitHandler(ctx, 10); err != nil {
			return err
		}

		// TODO make these separate commands, indexer should run on a single instance
		// and the processor can run on N instances since it pulls work from the queue.
		indexer.Start(ctx)
		processor.Start(ctx)

		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		<-ctx.Done()
		os.Exit(0)
		return nil
	},
}
