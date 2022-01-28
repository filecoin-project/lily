package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"gopkg.in/cheggaaa/pb.v1"

	fslock "github.com/ipfs/go-fs-lock"
	logging "github.com/ipfs/go-log/v2"
)

// Ported from github.com/filecoin-project/go-paramfetch

var log = logging.Logger("lily/vectors")

const gateway = "https://dweb.link/ipfs/"
const lockFile = "fetch.lock"
const vectordir = "/var/tmp/lily-test-vectors"
const vectordirenv = "LILY_TEST_VECTORS"
const lockRetry = time.Second * 10

func GetVectorDir() string {
	if os.Getenv(vectordirenv) == "" {
		return vectordir
	}
	return os.Getenv(vectordirenv)
}

var checked = map[string]struct{}{}
var checkedLk sync.Mutex

type VectorFile struct {
	Cid     string `json:"cid"`
	Network string `json:"network"`
	Digest  string `json:"digest"`
	From    int64  `json:"from"`
	To      int64  `json:"to"`
}

func GetVectors(ctx context.Context, vectorBytes []byte) error {
	if err := os.Mkdir(GetVectorDir(), 0755); err != nil && !os.IsExist(err) {
		return err
	}

	var testVectors map[string]VectorFile

	if err := json.Unmarshal(vectorBytes, &testVectors); err != nil {
		return err
	}

	ft := &fetch{}
	for vectorTar, info := range testVectors {
		// fetch each vector into its own directory,  I am sorry for the complexity here. This was added as a result of
		// chain snapshots needing to include a genesis file in addition to the snapshot since lotus lacks a long-lived testnet.
		vd := filepath.Base(vectorTar)
		vd = vd[0 : len(vd)-len(filepath.Ext(vd))]
		vectorDir := filepath.Join(GetVectorDir(), info.Network, vd)
		if err := os.MkdirAll(vectorDir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		ft.fetchAsync(ctx, vectorTar, vectorDir, info)
	}

	return ft.wait(ctx)
}

type fetch struct {
	wg      sync.WaitGroup
	fetchLk sync.Mutex

	errs []error
}

func (ft *fetch) fetchAsync(ctx context.Context, vectorTar, vectorDir string, info VectorFile) {
	ft.wg.Add(1)

	go func() {
		defer ft.wg.Done()

		path := filepath.Join(vectorDir, vectorTar)

		err := ft.checkFile(path, info)
		if !os.IsNotExist(err) && err != nil {
			log.Warn(err)
		}
		if err == nil {
			return
		}

		ft.fetchLk.Lock()
		defer ft.fetchLk.Unlock()

		var lockfail bool
		var unlocker io.Closer
		for {
			unlocker, err = fslock.Lock(GetVectorDir(), lockFile)
			if err == nil {
				break
			}

			lockfail = true

			le := fslock.LockedError("")
			if xerrors.As(err, &le) {
				log.Warnf("acquiring filesystem fetch lock: %s; will retry in %s", err, lockRetry)
				time.Sleep(lockRetry)
				continue
			}
			ft.errs = append(ft.errs, xerrors.Errorf("acquiring filesystem fetch lock: %w", err))
			return
		}
		defer func() {
			err := unlocker.Close()
			if err != nil {
				log.Errorw("unlock fs lock", "error", err)
			}
		}()
		if lockfail {
			// we've managed to get the lock, but we need to re-check file contents - maybe it's fetched now
			ft.fetchAsync(ctx, vectorTar, vectorDir, info)
			return
		}

		if err := doFetch(ctx, path, info); err != nil {
			ft.errs = append(ft.errs, xerrors.Errorf("fetching file %s failed: %w", path, err))
			return
		}
		ft.checkFile(path, info)
		if err != nil {
			ft.errs = append(ft.errs, xerrors.Errorf("checking file %s failed: %w", path, err))
			err := os.Remove(path)
			if err != nil {
				ft.errs = append(ft.errs, xerrors.Errorf("remove file %s failed: %w", path, err))
			}
		}
	}()
}

func (ft *fetch) wait(ctx context.Context) error {
	waitChan := make(chan struct{}, 1)

	go func() {
		defer close(waitChan)
		ft.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		log.Infof("context closed... shutting down")
	case <-waitChan:
		log.Infof("test vector fetching complete")
	}

	return multierr.Combine(ft.errs...)
}

func (ft *fetch) checkFile(path string, info VectorFile) error {
	checkedLk.Lock()
	_, ok := checked[path]
	checkedLk.Unlock()
	if ok {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	sum := h.Sum(nil)
	strSum := hex.EncodeToString(sum[:])
	if strSum == info.Digest {
		log.Infof("vector file %s is ok", path)

		checkedLk.Lock()
		checked[path] = struct{}{}
		checkedLk.Unlock()

		return nil
	}

	return xerrors.Errorf("checksum mismatch in test vector file %s, %s != %s", path, strSum, info.Digest)
}

func doFetch(ctx context.Context, out string, info VectorFile) error {
	gw := os.Getenv("IPFS_GATEWAY")
	if gw == "" {
		gw = gateway
	}
	log.Infof("Fetching %s from %s", out, gw)

	outf, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer outf.Close()

	fStat, err := outf.Stat()
	if err != nil {
		return err
	}
	header := http.Header{}
	header.Set("Range", "bytes="+strconv.FormatInt(fStat.Size(), 10)+"-")
	url, err := url.Parse(gw + info.Cid)
	if err != nil {
		return err
	}
	log.Infof("GET %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return err
	}
	req.Close = true
	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bar := pb.New64(fStat.Size() + resp.ContentLength)
	bar.Set64(fStat.Size())
	bar.Units = pb.U_BYTES
	bar.ShowSpeed = true
	bar.Start()

	_, err = io.Copy(outf, bar.NewProxyReader(resp.Body))

	bar.Finish()

	return err
}
