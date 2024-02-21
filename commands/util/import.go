package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/DataDog/zstd"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
	"gopkg.in/cheggaaa/pb.v1"

	"github.com/filecoin-project/lotus/chain/consensus"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/index"
	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/lotus/journal"
	"github.com/filecoin-project/lotus/journal/fsjournal"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/storage/sealer/ffiwrapper"
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
		return fmt.Errorf("failed to open blockstore: %w", err)
	}

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	j, err := fsjournal.OpenFSJournal(lr, journal.EnvDisabledEvents())
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}

	cst := store.NewChainStore(bs, bs, mds, filcns.Weight, j)
	defer cst.Close() //nolint:errcheck

	ts, _, err := cst.Import(ctx, fs)
	if err != nil {
		return fmt.Errorf("importing chain failed: %w", err)
	}

	if err := cst.FlushValidationCache(ctx); err != nil {
		return fmt.Errorf("flushing validation cache failed: %w", err)
	}

	gb, err := cst.GetTipsetByHeight(ctx, 0, ts, true)
	if err != nil {
		return err
	}

	err = cst.SetGenesis(ctx, gb.Blocks()[0])
	if err != nil {
		return err
	}

	stm, err := stmgr.NewStateManager(cst, consensus.NewTipSetExecutor(filcns.RewardFunc), vm.Syscalls(ffiwrapper.ProofVerifier), filcns.DefaultUpgradeSchedule(), nil, mds, index.DummyMsgIndex)
	if err != nil {
		return err
	}

	if !snapshot {
		log.Infof("validating imported chain...")
		if err := stm.ValidateChain(ctx, ts); err != nil {
			return fmt.Errorf("chain validation failed: %w", err)
		}
	}

	log.Infof("accepting %s as new head", ts.Cids())
	err = cst.ForceHeadSilent(ctx, ts)
	return err
}

func ImportChain(ctx context.Context, r repo.Repo, fname string, snapshot bool, backfillTipsetkeyRange int) (err error) {
	var rd io.Reader
	var l int64
	if strings.HasPrefix(fname, "http://") || strings.HasPrefix(fname, "https://") {
		resp, err := http.Get(fname) //nolint:gosec
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-200 response: %d", resp.StatusCode)
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
		return fmt.Errorf("failed to open blockstore: %w", err)
	}

	mds, err := lr.Datastore(context.TODO(), "/metadata")
	if err != nil {
		return err
	}

	j, err := fsjournal.OpenFSJournal(lr, journal.EnvDisabledEvents())
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}

	cst := store.NewChainStore(bs, bs, mds, filcns.Weight, j)
	defer cst.Close() //nolint:errcheck

	log.Infof("importing chain from %s...", fname)

	bufr := bufio.NewReaderSize(rd, 1<<20)

	header, err := bufr.Peek(4)
	if err != nil {
		return xerrors.Errorf("peek header: %w", err)
	}

	bar := pb.New64(l)
	br := bar.NewProxyReader(bufr)
	bar.ShowTimeLeft = true
	bar.ShowPercent = true
	bar.ShowSpeed = true
	bar.Units = pb.U_BYTES

	var ir io.Reader = br

	if string(header[1:]) == "\xB5\x2F\xFD" { // zstd
		zr := zstd.NewReader(br)
		defer func() {
			if err := zr.Close(); err != nil {
				log.Errorw("closing zstd reader", "error", err)
			}
		}()
		ir = zr
	}

	bar.Start()
	ts, _, err := cst.Import(ctx, ir)
	bar.Finish()

	if err != nil {
		return fmt.Errorf("importing chain failed: %w", err)
	}

	// The cst.Import function will only backfill 1800 epochs of tipsetkey,
	// Hence, the function is to backfill more epochs covered by the snapshot.
	err = backfillTipsetKey(ctx, ts, cst, backfillTipsetkeyRange)
	if err != nil {
		log.Errorf("backfill tipsetkey failed: %w", err)
	}

	if err := cst.FlushValidationCache(ctx); err != nil {
		return fmt.Errorf("flushing validation cache failed: %w", err)
	}

	gb, err := cst.GetTipsetByHeight(ctx, 0, ts, true)
	if err != nil {
		return err
	}

	err = cst.SetGenesis(ctx, gb.Blocks()[0])
	if err != nil {
		return err
	}

	stm, err := stmgr.NewStateManager(cst, consensus.NewTipSetExecutor(filcns.RewardFunc), vm.Syscalls(ffiwrapper.ProofVerifier), filcns.DefaultUpgradeSchedule(), nil, mds, index.DummyMsgIndex)
	if err != nil {
		return err
	}

	if !snapshot {
		log.Infof("validating imported chain...")
		if err := stm.ValidateChain(ctx, ts); err != nil {
			return fmt.Errorf("chain validation failed: %w", err)
		}
	}

	log.Infof("accepting %s as new head", ts.Cids())
	err = cst.ForceHeadSilent(ctx, ts)
	return err
}

func backfillTipsetKey(ctx context.Context, root *types.TipSet, cs *store.ChainStore, backfillRange int) (err error) {
	ts := root
	log.Infof("backfilling the tipsetkey into chainstore, attempt to backfill the last %v epochs starting from the head.", backfillRange)
	tssToPersist := make([]*types.TipSet, 0, backfillRange)
	for i := 0; i < backfillRange; i++ {
		tssToPersist = append(tssToPersist, ts)
		if err != nil {
			return
		}
		parentTsKey := ts.Parents()
		ts, err = cs.LoadTipSet(ctx, parentTsKey)
		if ts == nil || err != nil {
			log.Infof("Only able to load the last %d tipsets", i)
			break
		}
	}

	err = cs.PersistTipsets(ctx, tssToPersist)
	return
}
