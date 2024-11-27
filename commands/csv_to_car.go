package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/version"
	"github.com/urfave/cli/v2"

	mh "github.com/multiformats/go-multihash"
)

func init() {
	defaultName = "visor_" + version.String()
	hostname, err := os.Hostname()
	if err == nil {
		defaultName = fmt.Sprintf("%s_%s_%d", defaultName, hostname, os.Getpid())
	}
}

type CSVToCAROpts struct {
	CSVFile   string
	OutputCAR string
}

var CSVToCARFlags CSVToCAROpts

// Command flags for CSV-to-CAR conversion
var csvToCARFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "csv-file",
		EnvVars:     []string{"CSV_FILE"},
		Value:       "",
		Usage:       "Path to the input CSV file",
		Destination: &CSVToCARFlags.CSVFile,
	},
	&cli.StringFlag{
		Name:        "output-car",
		EnvVars:     []string{"OUTPUT_CAR"},
		Value:       "",
		Usage:       "Path to output the CAR file",
		Destination: &CSVToCARFlags.OutputCAR,
	},
}

// CSVToCAR Command for converting CSV file to CAR file
var CSVToCARCmd = &cli.Command{
	Name:  "csv-to-car",
	Usage: "Convert a CSV file to a CAR file",
	Flags: csvToCARFlags,
	Action: func(_ *cli.Context) error {
		bs, err := util.ReadCSVFile(CSVToCARFlags.CSVFile)
		if err != nil {
			return fmt.Errorf("failed to read CSV file: %w", err)
		}

		// Extract the filename from the CSV file path
		filename := filepath.Base(CSVToCARFlags.CSVFile)

		// Create the CAR file
		carData, err := util.MakeCar(filename, bs, mh.SHA2_256)
		if err != nil {
			return fmt.Errorf("failed to create CAR file: %w", err)
		}

		// Write the CAR file to disk
		if err := os.WriteFile(CSVToCARFlags.OutputCAR, carData, 0644); err != nil {
			return fmt.Errorf("failed to write CAR file: %w", err)
		}

		return nil
	},
}
