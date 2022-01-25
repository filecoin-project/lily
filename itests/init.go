package itests

import (
	"context"
	"encoding/json"
	"github.com/filecoin-project/lily/build"
	"github.com/filecoin-project/lily/itests/fetch"
	logging "github.com/ipfs/go-log/v2"
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
	ntwkDir := filepath.Join(fetch.GetVectorDir(), meta.Network)

	f, err := os.Open(filepath.Join(ntwkDir, fileName))
	if err != nil {
		return nil, err
	}
	return &TestVector{
		From: meta.From,
		To:   meta.To,
		File: f,
	}, nil
}
