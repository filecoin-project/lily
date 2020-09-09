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

		/*
			traceFn := metrics.InitTracer()
			defer traceFn()
		*/

		//ctx, api, closer, err := vapi.GetFullNodeAPI(cctx)
		//if err != nil {
		//return err
		//}
		//defer closer()

		db, err := storage.NewDatabase(ctx, cctx.String("db"))
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.CreateSchema(); err != nil {
			return err
		}

		mgr := services.NewServiceManager(api, db)
		if err := mgr.Run(ctx); err != nil {
			return err
		}

		logging.SetLogLevel("rpc", "info")

		<-ctx.Done()
		os.Exit(0)
		return nil
	},
}
