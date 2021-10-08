package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/filecoin-project/lily/schedule"
	"github.com/urfave/cli/v2"
)

func flagSet(fs ...[]cli.Flag) []cli.Flag {
	var flags []cli.Flag

	for _, f := range fs {
		flags = append(flags, f...)
	}

	return flags
}

func printNewJob(w io.Writer, res *schedule.JobSubmitResult) error {
	prettyJob, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", prettyJob); err != nil {
		return err
	}
	return nil
}
