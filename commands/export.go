package commands

import (
	"context"
	"io"
	"os"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/export"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type chainExportOps struct {
	repo        string
	max         uint64
	min         uint64
	outFile     string
	progress    bool
	includeMsgs bool
	includeRcpt bool
	includeStrt bool
}

var chainExportFlags chainExportOps

var ExportChainCmd = &cli.Command{
	Name:        "export",
	Description: "Export chain from repo (requires node to be offline)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "repo",
			Usage:       "the repo to export chain from",
			Value:       "~/.lily",
			Destination: &chainExportFlags.repo,
		},
		&cli.Uint64Flag{
			Name:        "max",
			Usage:       "inclusive max epoch to export",
			Required:    true,
			Destination: &chainExportFlags.max,
		},
		&cli.Uint64Flag{
			Name:        "min",
			Usage:       "inclusive min epoch to export",
			Required:    true,
			Destination: &chainExportFlags.min,
		},
		&cli.BoolFlag{
			Name:        "include-messages",
			Usage:       "exports messages if true",
			Value:       true,
			Destination: &chainExportFlags.includeMsgs,
		},
		&cli.BoolFlag{
			Name:        "include-receipts",
			Usage:       "exports receipts if true",
			Value:       true,
			Destination: &chainExportFlags.includeRcpt,
		},
		&cli.BoolFlag{
			Name:        "include-stateroots",
			Usage:       "exports stateroots if true",
			Value:       true,
			Destination: &chainExportFlags.includeStrt,
		},
		&cli.StringFlag{
			Name:        "out-file",
			Usage:       "file to export to",
			Value:       "chain_export.car",
			Destination: &chainExportFlags.outFile,
		},
		&cli.BoolFlag{
			Name:        "progress",
			Usage:       "set to show progress bar of export",
			Value:       true,
			Destination: &chainExportFlags.progress,
		},
	},
	Action: func(cctx *cli.Context) error {
		// use command context to allowing killing export at any point via ctrl+c
		ctx := cctx.Context

		// create file export will write to
		path, err := homedir.Expand(chainExportFlags.outFile)
		if err != nil {
			return err
		}
		outFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func() {
			if err := outFile.Close(); err != nil {
				log.Errorw("failed to close out file", "error", err.Error())
			}
		}()

		// open repo, blockstore, and chain store
		cs, bs, closer, err := openChainAndBlockStores(ctx, chainExportFlags.repo)
		if err != nil {
			return err
		}
		defer closer()

		log.Info("loading export head...")
		// get tipset at height `max` to start export from.
		exportHead, err := cs.GetTipsetByHeight(ctx, abi.ChainEpoch(chainExportFlags.max), cs.GetHeaviestTipSet(), true)
		if err != nil {
			return err
		}
		log.Infow("loaded export head", "tipset", exportHead.String())
		return export.NewChainExporter(exportHead, bs, outFile).
			Export(
				ctx,
				export.MinHeight(chainExportFlags.min),
				export.IncludeMessages(chainExportFlags.includeMsgs),
				export.IncludeReceipts(chainExportFlags.includeRcpt),
				export.IncludeStateRoots(chainExportFlags.includeStrt),
				export.WithProgress(chainExportFlags.progress),
			)
	},
}

func openChainAndBlockStores(ctx context.Context, path string) (*store.ChainStore, blockstore.Blockstore, func(), error) {
	repoDir, err := homedir.Expand(path)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("expand repo path (%s): %w", path, err)
	}

	r, err := repo.NewFS(repoDir)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("open repo (%s): %w", repoDir, err)
	}

	exists, err := r.Exists()
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("check repo (%s) exists: %w", repoDir, err)
	}
	if !exists {
		return nil, nil, nil, xerrors.Errorf("lily repo (%s) doesn't exists", repoDir)
	}

	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("lock repo (%s): %w", repoDir, err)
	}

	chainAndStateBs, err := lr.Blockstore(ctx, repo.UniversalBlockstore)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("accessing repo (%s) blockstore: %w", repoDir, err)
	}

	mds, err := lr.Datastore(ctx, "/metadata")
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("accessing repo (%s) datastore: %w", repoDir, err)
	}

	cs := store.NewChainStore(chainAndStateBs, chainAndStateBs, mds, nil, nil)
	if err := cs.Load(); err != nil {
		return nil, nil, nil, xerrors.Errorf("loading repo (%s) chain store: %w", repoDir, err)
	}

	return cs, chainAndStateBs,
		func() {
			if err := cs.Close(); err != nil {
				log.Errorw("failed to close chain store", "error", err.Error())
			}
			if c, ok := chainAndStateBs.(io.Closer); ok {
				if err := c.Close(); err != nil {
					log.Errorw("failed to close blockstore", "error", err.Error())
				}
			}
			if err := lr.Close(); err != nil {
				log.Errorw("failed to close locked repo", "error", err.Error())
			}
		},
		nil
}
