package commands

import (
	"fmt"
	"os"
	"path/filepath"

	mh "github.com/multiformats/go-multihash"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands/util"
)

type FileToCAROpts struct {
	TragetFile string
	OutputCAR  string
}

var FileToCARFlags FileToCAROpts

// Command flags for CSV-to-CAR conversion
var csvToCARFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "csv-file",
		EnvVars:     []string{"CSV_FILE"},
		Value:       "",
		Usage:       "Path to the input CSV file",
		Destination: &FileToCARFlags.TragetFile,
	},
	&cli.StringFlag{
		Name:        "output-car",
		EnvVars:     []string{"OUTPUT_CAR"},
		Value:       "",
		Usage:       "Path to output the CAR file",
		Destination: &FileToCARFlags.OutputCAR,
	},
}

// CSVToCAR Command for converting CSV file to CAR file
var FileToCARCmd = &cli.Command{
	Name:  "file-to-car",
	Usage: "Convert a file to a CAR file",
	Flags: csvToCARFlags,
	Action: func(_ *cli.Context) error {
		bs, err := util.ReadTargetFile(FileToCARFlags.TragetFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Extract the filename from the CSV file path
		filename := filepath.Base(FileToCARFlags.TragetFile)

		// Create the CAR file
		carData, err := util.MakeCar(filename, bs, mh.SHA2_256)
		if err != nil {
			return fmt.Errorf("failed to create CAR file: %w", err)
		}

		// Write the CAR file to disk
		if err := os.WriteFile(FileToCARFlags.OutputCAR, carData, 0644); err != nil {
			return fmt.Errorf("failed to write CAR file: %w", err)
		}

		return nil
	},
}
