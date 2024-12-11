package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/commands/util"
	"github.com/filecoin-project/lily/version"
)

func init() {
	defaultName = "visor_" + version.String()
	hostname, err := os.Hostname()
	if err == nil {
		defaultName = fmt.Sprintf("%s_%s_%d", defaultName, hostname, os.Getpid())
	}
}

type ImportFromCIDOpts struct {
	CID        string
	OutputCAR  string
	OutputFile string
}

var ImportFromCIDFlags ImportFromCIDOpts

// Command flags for CSV-to-CAR conversion
var flags = []cli.Flag{
	&cli.StringFlag{
		Name:        "cid",
		Value:       "",
		Usage:       "object cid",
		Destination: &ImportFromCIDFlags.CID,
	},
	&cli.StringFlag{
		Name:        "output-car",
		EnvVars:     []string{"OUTPUT_CAR"},
		Value:       "",
		Usage:       "Path to output the CAR file",
		Destination: &ImportFromCIDFlags.OutputCAR,
	},
	&cli.StringFlag{
		Name:        "output-file",
		Value:       "",
		Usage:       "Path to output the file",
		Destination: &ImportFromCIDFlags.OutputFile,
	},
}

// CSVToCAR Command for converting CSV file to CAR file
var ImportFromCIDCmd = &cli.Command{
	Name:  "cid-to-car",
	Usage: "Download cid to a CAR file",
	Flags: flags,
	Action: func(ctx *cli.Context) error {
		err := util.DownloadCarFile(context.TODO(), ctx, ImportFromCIDFlags.CID, ImportFromCIDFlags.OutputCAR)
		if err != nil {
			return err
		}
		err = util.ExtractCar(ImportFromCIDFlags.OutputCAR, ImportFromCIDFlags.OutputFile)
		if err != nil {
			fmt.Printf("got error in extract car: %v", err)
		}

		return nil
	},
}
