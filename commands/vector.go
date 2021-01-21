package commands

import (
	"context"
	"errors"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens/camera"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/filecoin-project/sentinel-visor/chain"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var Vector = &cli.Command{
	Name:  "vector",
	Usage: "Vector tooling for Visor.",
	Subcommands: []*cli.Command{
		BuildVector,
	},
}

var BuildVector = &cli.Command{
	Name:  "build",
	Usage: "Create a vector.",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:    "from",
			Usage:   "Limit actor and message processing to tipsets at or above `HEIGHT`",
			EnvVars: []string{"VISOR_HEIGHT_FROM"},
		},
		&cli.Int64Flag{
			Name:        "to",
			Usage:       "Limit actor and message processing to tipsets at or below `HEIGHT`",
			Value:       estimateCurrentEpoch(),
			DefaultText: "MaxInt64",
			EnvVars:     []string{"VISOR_HEIGHT_TO"},
		},
		&cli.StringFlag{
			Name:    "tasks",
			Usage:   "Comma separated list of tasks to run. Each task is reported separately in the database.",
			Value:   strings.Join([]string{chain.BlocksTask}, ","),
			EnvVars: []string{"VISOR_WALK_TASKS"},
		},
		&cli.StringFlag{
			Name:   "csv",
			Usage:  "Path to write csv files.",
			Hidden: true,
		},
	},
	Action: buildVector,
}

func buildVector(cctx *cli.Context) error {
	// Validate flags
	heightFrom := cctx.Int64("from")
	heightTo := cctx.Int64("to")

	if heightFrom > heightTo {
		return xerrors.Errorf("--from must not be greater than --to")
	}

	tasks := strings.Split(cctx.String("tasks"), ",")

	if err := setupLogging(cctx); err != nil {
		return xerrors.Errorf("setup logging: %w", err)
	}

	lensOpener, lensCloser, err := camera.NewCameraOpener(cctx, 10_000)
	if err != nil {
		return xerrors.Errorf("setup lens: %w", err)
	}
	defer func() {
		lensCloser()
	}()

	// TODO make this the default, possibly removing the flag since its more or less required.
	var strg model.Storage = &storage.NullStorage{}
	if cctx.String("csv") != "" {
		csvStorage, err := storage.NewCSVStorage(cctx.String("csv"))
		if err != nil {
			return xerrors.Errorf("new csv storage: %w", err)
		}
		strg = csvStorage
	} else {
		if cctx.String("db") == "" {
			log.Warnw("database not specified, data will not be persisted")
		} else {
			db, err := setupDatabase(cctx)
			if err != nil {
				return xerrors.Errorf("setup database: %w", err)
			}
			strg = db
		}
	}

	// Set up a context that is canceled when the command is interrupted
	ctx, cancel := context.WithCancel(cctx.Context)
	defer cancel()

	// Set up a signal handler to cancel the context
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
		}
	}()

	tsIndexer, err := chain.NewTipSetIndexer(lensOpener, strg, 0, cctx.String("name"), tasks)
	if err != nil {
		return xerrors.Errorf("setup indexer: %w", err)
	}
	defer func() {
		if err := tsIndexer.Close(); err != nil {
			log.Errorw("failed to close tipset indexer cleanly", "error", err)
		}
	}()
	walker := chain.NewWalker(tsIndexer, lensOpener, heightFrom, heightTo)

	err = walker.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	node, closer, err := lensOpener.Open(ctx)
	if err != nil {
		return xerrors.Errorf("open lens: %w", err)
	}
	defer closer()

	root, err := node.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(heightTo), types.EmptyTSK)
	if err != nil {
		return err
	}

	f, err := os.OpenFile("record.car", os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}

	if err := lensOpener.Record(ctx, f, root.Cids()...); err != nil {
		return err
	}

	return nil
}
