package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lily/schedule"
)

func FlagSet(fs ...[]cli.Flag) []cli.Flag {
	var flags []cli.Flag

	for _, f := range fs {
		flags = append(flags, f...)
	}

	return flags
}

func PrintNewJob(w io.Writer, res *schedule.JobSubmitResult) error {
	prettyJob, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", prettyJob); err != nil {
		return err
	}
	return nil
}
