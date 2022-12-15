package v8

import (
	"context"

	miner8 "github.com/filecoin-project/go-state-types/builtin/v8/miner"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func HandleV8MinerSectorChanges(ctx context.Context, changes []minerdiff.SectorChange) ([]*miner8.SectorOnChainInfo, error) {
	var sectors []*miner8.SectorOnChainInfo
	for _, change := range changes {
		if err := core.StateReadDeferred(ctx, *change.Current, func(sector *miner8.SectorOnChainInfo) error {
			sectors = append(sectors, sector)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return sectors, nil
}
