package sectorevent

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
)

type SectorEventType int64

const (
	PreCommitAdded SectorEventType = iota
	PreCommitExpired

	CommitCapacityAdded
	SectorAdded

	SectorExtended
	SectorSnapped

	SectorFaulted
	SectorRecovering
	SectorRecovered

	SectorExpired
	SectorTerminated
)

func (e SectorEventType) String() string {
	switch e {
	case PreCommitAdded:
		return "PRECOMMIT_ADDED"
	case PreCommitExpired:
		return "PRECOMMIT_EXPIRED"
	case CommitCapacityAdded:
		return "COMMIT_CAPACITY_ADDED"
	case SectorAdded:
		return "SECTOR_ADDED"
	case SectorExtended:
		return "SECTOR_EXTENDED"
	case SectorSnapped:
		return "SECTOR_SNAPPED"
	case SectorFaulted:
		return "SECTOR_FAULTED"
	case SectorRecovering:
		return "SECTOR_RECOVERING"
	case SectorRecovered:
		return "SECTOR_RECOVERED"
	case SectorExpired:
		return "SECTOR_EXPIRED"
	case SectorTerminated:
		return "SECTOR_TERMINATED"
	}
	panic(fmt.Sprintf("unhanded type %d developer error", e))
}

type SectorEvent struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
	Miner     address.Address
	SectorID  abi.SectorNumber
	Event     SectorEventType
}
