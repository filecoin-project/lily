package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/filecoin-project/monitor/lizard"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "monitor",
		Usage: "a tool for monitoring filecoin nodes",
		Description: `
~
 .             _.--._       /|
        .    .'()..()'.    / /
            ( '-^__^-' )  ( (    .
   .         \        /    \ \
       .      \      /      ) )        .
            .' -.__.- '.-.-'_.'
 .        .'  /-____-\  '.-'       .
          \  /-.____.-\  /-.
           \ \'-.__.-'/ /\|\|           .
          .'  '.    .'  '.
          |/\/\|    |/\/\|
`,
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			runCmd,
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
	}
}

type runOps struct {
	url      string
	database string
	schema   string
	name     string
	poolSize int
	lotus    string
}

var runFlags runOps

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start monitor service",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "url",
			Usage:       "specify postgres database connection string",
			Value:       "postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
			Destination: &runFlags.url,
		},
		&cli.StringFlag{
			Name:        "database",
			Usage:       "specify name of the database",
			Value:       "postgres",
			Destination: &runFlags.database,
		},
		&cli.StringFlag{
			Name:        "schema",
			Usage:       "specify postgres database schema name",
			Value:       "public",
			Destination: &runFlags.schema,
		},
		&cli.StringFlag{
			Name:        "name",
			Usage:       "specify name of application that will be used in database logs",
			Value:       "monitor",
			Destination: &runFlags.name,
		},
		&cli.IntFlag{
			Name:        "pool-size",
			Usage:       "Maximum number of socket connections.",
			Value:       10,
			Destination: &runFlags.poolSize,
		},
		&cli.StringFlag{
			Name:        "lotus",
			Usage:       "lotus connection string",
			Value:       "ws://localhost:1234",
			EnvVars:     []string{"FILADDR_LOTUS"},
			Destination: &runFlags.lotus,
		},
	},
	Action: func(cctx *cli.Context) error {
		// Set up a context that is canceled when the command is interrupted
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mapi := lizard.NewLizard(&lizard.Config{
			URL:      runFlags.url,
			Database: runFlags.database,
			Schema:   runFlags.schema,
			Name:     runFlags.name,
			PoolSize: runFlags.poolSize,
			LotusAPI: runFlags.lotus,
		})

		// setup api endpoints and connect to the database
		if err := mapi.Init(ctx); err != nil {
			return err
		}

		// Set up a signal handler to cancel the context
		go func() {
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
			select {
			case <-interrupt:
				cancel()
			}
		}()

		// api go brrr
		mapi.Start(ctx)
		<-ctx.Done()
		mapi.Stop()
		return nil
	},
}
