package commands

import (
	"time"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/urfave/cli/v2"
)

var mainnetGenesis = time.Date(2020, 8, 24, 22, 0, 0, 0, time.UTC)

func estimateCurrentEpoch() int64 {
	return int64(time.Since(mainnetGenesis) / (builtin.EpochDurationSeconds))
}

func flagSet(fs ...[]cli.Flag) []cli.Flag {
	var flags []cli.Flag

	for _, f := range fs {
		flags = append(flags, f...)
	}

	return flags
}
