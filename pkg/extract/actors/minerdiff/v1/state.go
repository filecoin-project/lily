package v1

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("extract/actors/miner")

type StateDiff struct {
	DiffMethods []actors.ActorStateDiff
}

func (s *StateDiff) State(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorDiffResult, error) {
	grp, grpctx := errgroup.WithContext(ctx)
	results, err := actors.ExecuteStateDiff(grpctx, grp, api, act, s.DiffMethods...)
	if err != nil {
		return nil, err
	}

	var stateDiff = new(StateDiffResult)
	for _, stateChange := range results {
		// some results may be nil, skip those
		if stateChange == nil {
			continue
		}
		switch stateChange.Kind() {
		case KindMinerInfo:
			stateDiff.InfoChange = stateChange.(*InfoChange)
		case KindMinerSector:
			stateDiff.SectorChanges = stateChange.(SectorChangeList)
		case KindMinerPreCommit:
			stateDiff.PreCommitChanges = stateChange.(PreCommitChangeList)
		case KindMinerSectorStatus:
			stateDiff.SectorStatusChanges = stateChange.(*SectorStatusChange)
		default:
			return nil, fmt.Errorf("unknown state change %s", stateChange.Kind())
		}
	}

	return stateDiff, nil
}

type StateDiffResult struct {
	InfoChange          *InfoChange
	PreCommitChanges    PreCommitChangeList
	SectorChanges       SectorChangeList
	SectorStatusChanges *SectorStatusChange
}

func (s *StateDiffResult) Kind() string {
	return "miner"
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, s store.Store) (typegen.CBORMarshaler, error) {
	out := &StateChange{}

	if sectors := sd.SectorChanges; sectors != nil {
		root, err := sectors.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.Sectors = &root
	}

	if precommits := sd.PreCommitChanges; precommits != nil {
		root, err := precommits.ToAdtMap(s, 5)
		if err != nil {
			return nil, err
		}
		out.PreCommits = &root
	}

	if info := sd.InfoChange; info != nil {
		c, err := s.Put(ctx, info)
		if err != nil {
			return nil, err
		}
		out.Info = &c
	}

	if sectorstatus := sd.SectorStatusChanges; sectorstatus != nil {
		c, err := s.Put(ctx, sectorstatus)
		if err != nil {
			return nil, err
		}
		out.SectorStatus = &c
	}
	return out, nil
}

type StateChange struct {
	// SectorStatus is the sectors whose status changed for this miner or empty.
	SectorStatus *cid.Cid `cborgen:"sector_status"`
	// Info is the cid of the miner change info that changed for this miner or empty.
	Info *cid.Cid `cborgen:"info"`
	// PreCommits is an HAMT of the pre commits that changed for this miner.
	PreCommits *cid.Cid `cborgen:"pre_commits"`
	// Sectors is an HAMT of the sectors that changed for this miner.
	Sectors *cid.Cid `cborgen:"sectors"`
}

func (sc *StateChange) ToStateDiffResult(ctx context.Context, s store.Store) (*StateDiffResult, error) {
	out := &StateDiffResult{
		InfoChange:          nil,
		SectorStatusChanges: nil,
		SectorChanges:       SectorChangeList{},
		PreCommitChanges:    PreCommitChangeList{},
	}

	//
	// SectorChangeList
	if sc.Sectors != nil {
		sectorMap, err := adt.AsMap(s, *sc.Sectors, 5)
		if err != nil {
			return nil, err
		}

		sectors := SectorChangeList{}
		sectorChange := new(SectorChange)
		if err := sectorMap.ForEach(sectorChange, func(sectorNumber string) error {
			val := new(SectorChange)
			*val = *sectorChange
			sectors = append(sectors, val)
			return nil
		}); err != nil {
			return nil, err
		}
		out.SectorChanges = sectors
	}

	//
	// PrecommitChangeList

	if sc.PreCommits != nil {
		precommitMap, err := adt.AsMap(s, *sc.PreCommits, 5)
		if err != nil {
			return nil, err
		}

		precommits := PreCommitChangeList{}
		precommitChange := new(PreCommitChange)
		if err := precommitMap.ForEach(precommitChange, func(sectorNumber string) error {
			val := new(PreCommitChange)
			*val = *precommitChange
			precommits = append(precommits, val)
			return nil
		}); err != nil {
			return nil, err
		}
		out.PreCommitChanges = precommits
	}

	//
	// Info
	if sc.Info != nil {
		info := new(InfoChange)
		if err := s.Get(ctx, *sc.Info, info); err != nil {
			return nil, err
		}
		out.InfoChange = info
	}

	//
	// SectorStatus

	if sc.SectorStatus != nil {
		status := new(SectorStatusChange)
		if err := s.Get(ctx, *sc.SectorStatus, status); err != nil {
			return nil, err
		}
		out.SectorStatusChanges = status
	}

	return out, nil
}
