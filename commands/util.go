package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/schedule"
	"github.com/filecoin-project/lily/storage"
)

func FlagSet(fs ...[]cli.Flag) []cli.Flag {
	var flags []cli.Flag

	for _, f := range fs {
		flags = append(flags, f...)
	}

	return flags
}

func PrintNewJob(w io.Writer, res *schedule.JobSubmitResult) error {
	prettyJob, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", prettyJob); err != nil {
		return err
	}
	return nil
}

func SetupContextWithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func SetupStorage(configPath string, storageStr string) (strg model.Storage, err error) {
	if storageStr != "" {
		cfg, err := config.FromFile(configPath)
		if err != nil {
			return nil, err
		}

		md := storage.Metadata{
			JobName: storageStr,
		}

		// context for db connection
		ctxDB := context.Background()

		sc, err := storage.NewCatalog(cfg.Storage)
		if err != nil {
			return nil, err
		}
		strg, err = sc.Connect(ctxDB, syncFlags.storage, md)
		if err != nil {
			return nil, err
		}
	}

	return strg, nil
}
