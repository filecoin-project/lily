package v0

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
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
	grp, grpCtx := errgroup.WithContext(ctx)
	results := make(chan actors.ActorStateChange, len(s.DiffMethods))

	for _, f := range s.DiffMethods {
		f := f
		grp.Go(func() error {
			stateChange, err := f.Diff(grpCtx, api, act)
			if err != nil {
				return err
			}

			// TODO maybe this method should also return a bool to indicate if anything actually changed, instead of two null values.
			if stateChange != nil {
				results <- stateChange
			}
			return nil
		})
	}

	go func() {
		if err := grp.Wait(); err != nil {
			log.Error(err)
		}
		close(results)
	}()
	var stateDiff = new(StateDiffResult)
	for stateChange := range results {
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

type StateChange struct {
	// SectorStatus is the sectors whose status changed for this miner or empty.
	SectorStatus *cid.Cid `cborgen:"sector_status"`
	// Info is the cid of the miner change info that changed for this miner or empty.
	Info *cid.Cid `cborgen:"info"`
	// PreCommits is an HAMT of the pre commits that changed for this miner.
	PreCommits cid.Cid `cborgen:"pre_commits"`
	// Sectors is an HAMT of the sectors that changed for this miner.
	Sectors cid.Cid `cborgen:"sectors"`
}

func DecodeStateDiffResultFromStateChange(ctx context.Context, bs blockstore.Blockstore, sc *StateChange) (*StateDiffResult, error) {
	out := &StateDiffResult{}
	adtStore := store.WrapBlockStore(ctx, bs)

	//
	// SectorChangeList
	{
		sectorChangeList := &SectorChangeList{}
		if err := sectorChangeList.FromAdtMap(adtStore, sc.Sectors, 5); err != nil {
			return nil, err
		}
		out.SectorChanges = *sectorChangeList
		/*
			sectorMap, err := adt.AsMap(adtStore, sc.Sectors, 5)
			if err != nil {
				return nil, err
			}

			sectorChangeList := SectorChangeList{}
			sectorChange := new(SectorChange)
			if err := sectorMap.ForEach(sectorChange, func(sectorNumber string) error {
				val := new(SectorChange)
				*val = *sectorChange
				sectorChangeList = append(sectorChangeList, val)
				return nil
			}); err != nil {
				return nil, err
			}
			out.SectorChanges = sectorChangeList

		*/
	}

	//
	// PrecommitChangeList
	{
		precommitChangeList := &PreCommitChangeList{}
		if err := precommitChangeList.FromAdtMap(adtStore, sc.PreCommits, 5); err != nil {
			return nil, err
		}
		out.PreCommitChanges = *precommitChangeList
		/*
			preCommitMap, err := adt.AsMap(adtStore, sc.PreCommits, 5)
			if err != nil {
				return nil, err
			}

			preCommitChangeList := PreCommitChangeList{}
			preCommitChange := new(PreCommitChange)
			if err := preCommitMap.ForEach(preCommitChange, func(sectorNumber string) error {
				val := new(PreCommitChange)
				*val = *preCommitChange
				preCommitChangeList = append(preCommitChangeList, val)
				return nil
			}); err != nil {
				return nil, err
			}
			out.PreCommitChanges = preCommitChangeList

		*/
	}

	//
	// Info
	{
		if sc.Info != nil {
			blk, err := bs.Get(ctx, *sc.Info)
			if err != nil {
				return nil, err
			}
			info, err := DecodeInfo(blk.RawData())
			if err != nil {
				return nil, err
			}
			out.InfoChange = info
		}
	}

	//
	// SectorStatus
	{

		if sc.SectorStatus != nil {
			blk, err := bs.Get(ctx, *sc.SectorStatus)
			if err != nil {
				return nil, err
			}
			ss, err := DecodeSectorStatus(blk.RawData())
			if err != nil {
				return nil, err
			}
			out.SectorStatusChanges = ss
		}
	}

	return out, nil
}

func (sd *StateDiffResult) MarshalStateChange(ctx context.Context, bs blockstore.Blockstore) (typegen.CBORMarshaler, error) {
	out := &StateChange{}
	adtStore := store.WrapBlockStore(ctx, bs)

	if sectors := sd.SectorChanges; sectors != nil {
		root, err := sectors.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.Sectors = root
	}

	if precommits := sd.PreCommitChanges; precommits != nil {
		root, err := precommits.ToAdtMap(adtStore, 5)
		if err != nil {
			return nil, err
		}
		out.PreCommits = root
	}

	if info := sd.InfoChange; info != nil {
		blk, err := info.ToStorageBlock()
		if err != nil {
			return nil, err
		}
		if err := bs.Put(ctx, blk); err != nil {
			return nil, err
		}
		c := blk.Cid()
		out.Info = &c
	}

	if sectorstatus := sd.SectorStatusChanges; sectorstatus != nil {
		blk, err := sectorstatus.ToStorageBlock()
		if err != nil {
			return nil, err
		}
		if err := bs.Put(ctx, blk); err != nil {
			return nil, err
		}
		c := blk.Cid()
		out.SectorStatus = &c
	}
	return out, nil
}
