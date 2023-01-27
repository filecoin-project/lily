package types

import (
	"github.com/filecoin-project/go-address"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"
)

type MinerStateChange struct {
	Address     address.Address
	StateChange *minerdiff.StateDiffResult
}
