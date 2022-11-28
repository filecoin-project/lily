package minertransform

import (
	"bytes"
	"context"
	"sort"

	"github.com/filecoin-project/go-state-types/builtin/v8/util/adt"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/util"
)

type MinerIPLDChangeContainer struct {
	Sectors    *adt.Array
	PreCommits *adt.Array
	Info       []byte
}

func V0MinerHandler(ctx context.Context, stateDiff *minerdiff.StateDiff) (*MinerIPLDChangeContainer, error) {
	var (
		err        error
		sectors    *adt.Array
		preCommits *adt.Array
		info       []byte
	)
	if stateDiff.InfoChange != nil {
		info, err = V0MinerInfoHandler(ctx, stateDiff.InfoChange)
		if err != nil {
			return nil, err
		}
	}
	if stateDiff.SectorChanges != nil {
		sectors, err = V0MinerSectorHandler(ctx, stateDiff.SectorChanges)
		if err != nil {
			return nil, err
		}
	}
	if stateDiff.PreCommitChanges != nil {
		preCommits, err = V0MinerPreCommitHandler(ctx, stateDiff.PreCommitChanges)
		if err != nil {
			return nil, err
		}
	}
	return &MinerIPLDChangeContainer{
		Sectors:    sectors,
		PreCommits: preCommits,
		Info:       info,
	}, nil
}

func V0MinerPreCommitHandler(ctx context.Context, preCommitChanges minerdiff.PreCommitChangeList) (*adt.Array, error) {
	// deserialize the deferred state to concrete type
	minerPreCommits := make([]*miner0.SectorPreCommitOnChainInfo, len(preCommitChanges))
	for i, preCommit := range preCommitChanges {
		var minerPreCommit *miner0.SectorPreCommitOnChainInfo
		if err := minerPreCommit.UnmarshalCBOR(bytes.NewReader(preCommit.PreCommit.Raw)); err != nil {
			return nil, err
		}
		minerPreCommits[i] = minerPreCommit
	}
	// sort the resulting list for deterministic ordering while inserting to AMT
	sort.Slice(minerPreCommits, func(i, j int) bool {
		ic, err := util.CidOf(minerPreCommits[i])
		if err != nil {
			panic(err)
		}
		jc, err := util.CidOf(minerPreCommits[j])
		if err != nil {
			panic(err)
		}
		return ic.KeyString() < jc.KeyString()
	})
	// define the array and insert all
	arr, err := adt.MakeEmptyArray(nil, 5)
	if err != nil {
		return nil, err
	}
	for _, c := range minerPreCommits {
		if err := arr.AppendContinuous(c); err != nil {
			return nil, err
		}
	}
	return arr, nil
}

func V0MinerSectorHandler(ctx context.Context, sectorChanges minerdiff.SectorChangeList) (*adt.Array, error) {
	minerSectors := make([]*miner0.SectorOnChainInfo, len(sectorChanges))
	for i, sector := range sectorChanges {
		var minerSector *miner0.SectorOnChainInfo
		if err := minerSector.UnmarshalCBOR(bytes.NewReader(sector.Sector.Raw)); err != nil {
			return nil, err
		}
		minerSectors[i] = minerSector
	}
	// sort the resulting list for deterministic ordering while inserting to AMT
	sort.Slice(minerSectors, func(i, j int) bool {
		ic, err := util.CidOf(minerSectors[i])
		if err != nil {
			panic(err)
		}
		jc, err := util.CidOf(minerSectors[j])
		if err != nil {
			panic(err)
		}
		return ic.KeyString() < jc.KeyString()
	})
	// define the array and insert all
	arr, err := adt.MakeEmptyArray(nil, 5)
	if err != nil {
		return nil, err
	}
	for _, c := range minerSectors {
		if err := arr.AppendContinuous(c); err != nil {
			return nil, err
		}
	}
	return arr, nil
}

func V0MinerInfoHandler(ctx context.Context, infoChange *minerdiff.InfoChange) ([]byte, error) {
	var minerInfo miner0.MinerInfo
	if err := minerInfo.UnmarshalCBOR(bytes.NewReader(infoChange.Info.Raw)); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := minerInfo.MarshalCBOR(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
