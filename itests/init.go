package itests

import (
	"archive/tar"
	"context"
	"encoding/json"
	"github.com/filecoin-project/lily/build"
	"github.com/filecoin-project/lily/itests/fetch"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"io"
	"os"
	"path/filepath"
	"time"
)

var log = logging.Logger("lily/itests")

const (
	Calibnet = "calibnet"
	Mainnet  = "mainnet"
)

type TestVector struct {
	From, To int64
	File     *os.File
	Snapshot *os.File
	Genesis  *os.File
}

func (tv *TestVector) Close() error {
	var errs []error
	if err := tv.Genesis.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := tv.Snapshot.Close(); err != nil {
		errs = append(errs, err)
	}
	return multierr.Combine(errs...)
}

var CalibnetTestVectors []*TestVector
var MainnetTestVectors []*TestVector

func init() {
	// Attempt to download all test vectors in parallel in 3 mins
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
	defer cancel()
	if err := fetch.GetVectors(ctx, build.VectorsJSON()); err != nil {
		log.Fatalf("fetching test vectors: %v", err)
	}

	// build lists of test vectors corresponding to networks; to be used in tests
	var testVectors map[string]fetch.VectorFile
	if err := json.Unmarshal(build.VectorsJSON(), &testVectors); err != nil {
		log.Fatal(err)
	}

	calibVectorCount, mainVectorCount := 0, 0
	for filename, meta := range testVectors {
		switch meta.Network {
		case Calibnet:
			metaVector, err := vectorsForNetwork(filename, meta)
			if err != nil {
				log.Fatal(err)
			}
			CalibnetTestVectors = append(CalibnetTestVectors, metaVector)
			calibVectorCount++
		case Mainnet:
			metaVector, err := vectorsForNetwork(filename, meta)
			if err != nil {
				log.Fatal(err)
			}
			MainnetTestVectors = append(MainnetTestVectors, metaVector)
			mainVectorCount++
		}
	}
	// this check ensures all the vectors in fetch.VectorFile were downloaded and will be run in the testing suite.
	if len(MainnetTestVectors) != mainVectorCount {
		log.Fatal("Failed to download all expected test vectors for mainnet")
	}
	if len(CalibnetTestVectors) != calibVectorCount {
		log.Fatal("Failed to download all expected test vectors for calibnet")
	}
}

func vectorsForNetwork(fileName string, meta fetch.VectorFile) (*TestVector, error) {
	vd := filepath.Base(fileName)
	vd = vd[0 : len(vd)-len(filepath.Ext(vd))]
	vectorDir := filepath.Join(fetch.GetVectorDir(), meta.Network, vd)

	vectorTar := filepath.Join(vectorDir, fileName)

	f, err := os.Open(vectorTar)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tareader := tar.NewReader(f)
	var genesis, snapshot *os.File
	for {
		t, err := tareader.Next()
		if err == io.EOF {
			break
		}
		switch t.Name {
		case "snapshot.car":
			snapshot, err = os.Create(filepath.Join(vectorDir, t.Name))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(snapshot, tareader); err != nil {
				return nil, err
			}
			if _, err := snapshot.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
		case "genesis.car":
			genesis, err = os.Create(filepath.Join(vectorDir, t.Name))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(genesis, tareader); err != nil {
				return nil, err
			}
			if _, err := genesis.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
		default:
			return nil, xerrors.Errorf("unexpected file: %v", t.Name)
		}
	}

	return &TestVector{
		From:     meta.From,
		To:       meta.To,
		File:     f,
		Genesis:  genesis,
		Snapshot: snapshot,
	}, nil
}
