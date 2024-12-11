package util

import (
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car/cmd/car/lib"
	"github.com/ipld/go-car/v2"
	carstorage "github.com/ipld/go-car/v2/storage"
	"github.com/ipld/go-car/v2/storage/deferred"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	pstorage "github.com/ipld/go-ipld-prime/storage"
	trustlessutils "github.com/ipld/go-trustless-utils"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lassie/pkg/lassie"
	"github.com/filecoin-project/lassie/pkg/storage"
	"github.com/filecoin-project/lassie/pkg/types"
)

func DownloadCarFile(ctx context.Context, cctx *cli.Context, cidStr, outputDir string) error {
	// Create a default lassie instance
	lassie, err := lassie.NewLassie(ctx)
	if err != nil {
		return err
	}

	// Prepare the fetch
	rootCid := cid.MustParse(cidStr) // The CID to fetch
	tempDir := cctx.String("tempdir")
	store := storage.NewDeferredStorageCar(tempDir, rootCid) // The place to put the CAR file

	var carWriter storage.DeferredWriter
	carOpts := []car.Option{
		car.WriteAsCarV1(true),
		car.StoreIdentityCIDs(false),
		car.UseWholeCIDs(false),
	}

	carWriter = deferred.NewDeferredCarWriterForPath(outputDir, []cid.Cid{rootCid}, carOpts...)

	carStore := storage.NewCachingTempStore(carWriter.BlockWriteOpener(), store)

	request, err := types.NewRequestForPath(carStore, rootCid, "", trustlessutils.DagScopeAll, nil) // The fetch request
	if err != nil {
		return err
	}

	// Fetch the CID
	stats, err := lassie.Fetch(ctx, request)
	if err != nil {
		return err
	}

	// Print the stats
	fmt.Printf("Fetched %d blocks in %d bytes, filename: %v\n", stats.Blocks, stats.Size, outputDir)

	return nil
}

// ExtractCar extracts files from a CAR file to the specified output directory.
func ExtractCar(inputFilePath string, outputDir string) error {
	var store pstorage.ReadableStorage
	var roots []cid.Cid

	carFile, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("opne car file error")
		return err
	}
	store, err = carstorage.OpenReadable(carFile)
	if err != nil {
		fmt.Printf("opne store error")
		return err
	}
	roots = store.(carstorage.ReadableCar).Roots()

	ls := cidlink.DefaultLinkSystem()
	ls.TrustedStorage = true
	ls.SetReadStorage(store)

	path := []string{}

	var extractedFiles int
	for _, root := range roots {
		count, err := lib.ExtractToDir(context.TODO(), &ls, root, outputDir, path, false, nil)
		if err != nil {
			fmt.Printf("write error...")
			return err
		}
		extractedFiles += count
	}
	if extractedFiles == 0 {
		return cli.Exit("no files extracted", 1)
	}

	fmt.Printf("extracted %d file(s)\n", extractedFiles)

	return nil
}
