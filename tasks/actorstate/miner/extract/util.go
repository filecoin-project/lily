package extract

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
)

func NewDiffCache() *DiffCache {
	return &DiffCache{cache: make(map[diffType]interface{})}
}

type DiffCache struct {
	cache map[diffType]interface{}
}

type diffType string

const (
	PreCommitDiff diffType = "PRECOMMIT"
	SectorDiff    diffType = "SECTOR"
)

func (d *DiffCache) Put(diff diffType, result interface{}) {
	d.cache[diff] = result
}

func (d *DiffCache) Get(diffType diffType) (interface{}, bool) {
	result, found := d.cache[diffType]
	return result, found
}

func GetPreCommitDiff(ctx context.Context, ec *MinerStateExtractionContext) (*miner.PreCommitChanges, error) {
	preCommitChanges := new(miner.PreCommitChanges)
	result, found := ec.Cache.Get(PreCommitDiff)
	if !found {
		var err error
		preCommitChanges, err = miner.DiffPreCommits(ctx, ec.Store, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		ec.Cache.Put(PreCommitDiff, preCommitChanges)
	} else {
		// a nil diff is a valid result, we want to keep this as to avoid rediffing to get nil
		if result == nil {
			return preCommitChanges, nil
		}
		preCommitChanges = result.(*miner.PreCommitChanges)
	}
	return preCommitChanges, nil
}

func GetSectorDiff(ctx context.Context, ec *MinerStateExtractionContext) (*miner.SectorChanges, error) {
	sectorChanges := new(miner.SectorChanges)
	result, found := ec.Cache.Get(SectorDiff)
	if !found {
		var err error
		sectorChanges, err = miner.DiffSectors(ctx, ec.Store, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		ec.Cache.Put(SectorDiff, sectorChanges)
	} else {
		// a nil diff is a valid result, we want to keep this as to avoid rediffing to get nil
		if result == nil {
			return sectorChanges, nil
		}
		sectorChanges = result.(*miner.SectorChanges)
	}
	return sectorChanges, nil
}
