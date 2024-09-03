package parsedmessage

import (
	"encoding/json"
	"fmt"

	// Other necessary imports, possibly including types from the Lily project

	"github.com/filecoin-project/go-bitfield"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v14/miner"
)

func parseParamsInDetail(method string, params string) ([]uint64, error) {
	sectorNumbers := []uint64{}

	switch method {
	case "ProveCommitAggregate":
		var aggregateParams minertypes.ProveCommitAggregateParams
		if err := json.Unmarshal([]byte(params), &aggregateParams); err != nil {
			return sectorNumbers, err
		}
		// Assuming AggregateProveCommitParams has a field SectorNumbers which is a slice
		sectorNumbers, _ = aggregateParams.SectorNumbers.All(bitfield.MaxEncodedSize)

	case "ProveCommitSector":
		var sectorParams minertypes.ProveCommitSectorParams
		if err := json.Unmarshal([]byte(params), &sectorParams); err != nil {
			return sectorNumbers, err
		}
		sectorNumbers = []uint64{uint64(sectorParams.SectorNumber)}

	case "ProveCommitSectors3":
		var sectors3Params minertypes.ProveCommitSectors3Params
		if err := json.Unmarshal([]byte(params), &sectors3Params); err != nil {
			return sectorNumbers, err
		}
		// Assuming ProveCommitSectors3Params has a field SectorNumbers which is a slice
		if len(sectors3Params.SectorActivations) > 0 {
			for _, sector := range sectors3Params.SectorActivations {
				sectorNumbers = append(sectorNumbers, uint64(sector.SectorNumber))
			}
		}

	case "ProveCommitSectorsNI":
		var sectorsNIParams minertypes.ProveCommitSectorsNIParams
		if err := json.Unmarshal([]byte(params), &sectorsNIParams); err != nil {
			return sectorNumbers, err
		}
		// Assuming ProveCommitSectorsNIParams has a field SectorNumbers which is a slice
		if len(sectorsNIParams.Sectors) > 0 {
			for _, sector := range sectorsNIParams.Sectors {
				sectorNumbers = append(sectorNumbers, uint64(sector.SealerID))
			}
		}

	default:
		return sectorNumbers, fmt.Errorf("unsupported method: %s", method)
	}

	return sectorNumbers, nil
}
