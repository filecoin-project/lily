package commands

import (
	"time"

	"github.com/filecoin-project/lotus/chain/actors/builtin"
)

var mainnetGenesis = time.Date(2020, 8, 24, 22, 0, 0, 0, time.UTC)

func estimateCurrentEpoch() int64 {
	return int64(time.Since(mainnetGenesis) / (builtin.EpochDurationSeconds))
}
