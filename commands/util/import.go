package util

import (
	"bufio"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/journal/fsjournal"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
	"gopkg.in/cheggaaa/pb.v1"
)

// used for test vectors only
func ImportFromFsFile(ctx context.Context, r repo.Repo, fs fs.File, snapshot bool) (err error) {
	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	bs, err := lr.Blockstore(ctx, repo.UniversalBlockstore)
	if err != nil {
		return xerrors.Errorf("failed to open blockstore: %w", err)
	}

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	j, err := fsjournal.OpenFSJournal(lr, journal.EnvDisabledEvents())
	if err != nil {
		return xerrors.Errorf("failed to open journal: %w", err)
	}

	cst := store.NewChainStore(bs, bs, mds, filcns.Weight, j)
	defer cst.Close() //nolint:errcheck

	ts, err := cst.Import(fs)

	if err != nil {
		return xerrors.Errorf("importing chain failed: %w", err)
	}

	if err := cst.FlushValidationCache(); err != nil {
		return xerrors.Errorf("flushing validation cache failed: %w", err)
	}

	gb, err := cst.GetTipsetByHeight(ctx, 0, ts, true)
	if err != nil {
		return err
	}

	err = cst.SetGenesis(gb.Blocks()[0])
	if err != nil {
		return err
	}

	stm, err := stmgr.NewStateManager(cst, filcns.NewTipSetExecutor(), vm.Syscalls(ffiwrapper.ProofVerifier), filcns.DefaultUpgradeSchedule(), nil)
	if err != nil {
		return err
	}

	if !snapshot {
		log.Infof("validating imported chain...")
		if err := stm.ValidateChain(ctx, ts); err != nil {
			return xerrors.Errorf("chain validation failed: %w", err)
		}
	}

	log.Infof("accepting %s as new head", ts.Cids())
	if err := cst.ForceHeadSilent(ctx, ts); err != nil {
		return err
	}

	return nil
}

func ImportChain(ctx context.Context, r repo.Repo, fname string, snapshot bool) (err error) {
	var rd io.Reader
	var l int64
	if strings.HasPrefix(fname, "http://") || strings.HasPrefix(fname, "https://") {
		resp, err := http.Get(fname) //nolint:gosec
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("non-200 response: %d", resp.StatusCode)
		}

		rd = resp.Body
		l = resp.ContentLength
	} else {
		fname, err = homedir.Expand(fname)
		if err != nil {
			return err
		}

		fi, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer fi.Close() //nolint:errcheck

		st, err := os.Stat(fname)
		if err != nil {
			return err
		}

		rd = fi
		l = st.Size()
	}

	lr, err := r.Lock(repo.FullNode)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	bs, err := lr.Blockstore(ctx, repo.UniversalBlockstore)
	if err != nil {
		return xerrors.Errorf("failed to open blockstore: %w", err)
	}

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	j, err := fsjournal.OpenFSJournal(lr, journal.EnvDisabledEvents())
	if err != nil {
		return xerrors.Errorf("failed to open journal: %w", err)
	}

	cst := store.NewChainStore(bs, bs, mds, filcns.Weight, j)
	defer cst.Close() //nolint:errcheck

	log.Infof("importing chain from %s...", fname)

	bufr := bufio.NewReaderSize(rd, 1<<20)

	bar := pb.New64(l)
	br := bar.NewProxyReader(bufr)
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.ShowSpeed = true
	bar.Units = pb.U_BYTES

	bar.Start()
	ts, err := cst.Import(br)
	bar.Finish()

	if err != nil {
		return xerrors.Errorf("importing chain failed: %w", err)
	}

	if err := cst.FlushValidationCache(); err != nil {
		return xerrors.Errorf("flushing validation cache failed: %w", err)
	}

	gb, err := cst.GetTipsetByHeight(ctx, 0, ts, true)
	if err != nil {
		return err
	}

	err = cst.SetGenesis(gb.Blocks()[0])
	if err != nil {
		return err
	}

	stm, err := stmgr.NewStateManager(cst, filcns.NewTipSetExecutor(), vm.Syscalls(ffiwrapper.ProofVerifier), filcns.DefaultUpgradeSchedule(), nil)
	if err != nil {
		return err
	}

	if !snapshot {
		log.Infof("validating imported chain...")
		if err := stm.ValidateChain(ctx, ts); err != nil {
			return xerrors.Errorf("chain validation failed: %w", err)
		}
	}

	log.Infof("accepting %s as new head", ts.Cids())
	if err := cst.ForceHeadSilent(ctx, ts); err != nil {
		return err
	}

	return nil
}
