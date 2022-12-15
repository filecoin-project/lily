package miner

import (
	"context"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	cborminer "github.com/filecoin-project/lily/pkg/transform/cbor/miner"
)

func DecodeMinerStateDiff(ctx context.Context, store adt.Store, change cborminer.StateChange) (*minerdiff.StateDiff, error) {
	out := &minerdiff.StateDiff{
		PreCommitChanges: minerdiff.PreCommitChangeList{},
		SectorChanges:    minerdiff.SectorChangeList{},
		SectorStatusChanges: &minerdiff.SectorStatusChange{
			Removed:    bitfield.New(),
			Recovering: bitfield.New(),
			Faulted:    bitfield.New(),
			Recovered:  bitfield.New(),
		},
	}
	if c := change.Info; c != nil {
		info, err := DecodeMinerInfoChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.InfoChange = info
	}

	if c := change.Funds; c != nil {
		funds, err := DecodeMinerFundsChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.FundsChange = funds
	}

	if c := change.Debt; c != nil {
		debt, err := DecodeMinerDebtChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.DebtChange = debt
	}

	if c := change.SectorStatus; c != nil {
		sectorStatus, err := DecodeMinerSectorStatusChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.SectorStatusChanges = sectorStatus
	}

	if c := change.Sectors; c != nil {
		sectors, err := DecodeMinerSectorChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.SectorChanges = sectors
	}

	if c := change.PreCommits; c != nil {
		precommits, err := DecodeMinerPreCommitChanges(ctx, store, *c)
		if err != nil {
			return nil, err
		}
		out.PreCommitChanges = precommits
	}
	return out, nil
}

// TODO dubious on usage of pointers in the decode methods.

func DecodeMinerInfoChanges(ctx context.Context, store adt.Store, info cid.Cid) (*minerdiff.InfoChange, error) {
	var infoContainer cborminer.Info
	if err := store.Get(ctx, info, &infoContainer); err != nil {
		return nil, err
	}
	var minfo typegen.Deferred
	if err := store.Get(ctx, infoContainer.Info, &minfo); err != nil {
		return nil, err
	}
	return &minerdiff.InfoChange{
		// TODO this seems error prone, bad copy
		Info:   &minfo,
		Change: infoContainer.Change,
	}, nil
}

func DecodeMinerFundsChanges(ctx context.Context, store adt.Store, funds cid.Cid) (*minerdiff.FundsChange, error) {
	minerFundsChange := new(minerdiff.FundsChange)
	if err := store.Get(ctx, funds, minerFundsChange); err != nil {
		return nil, err
	}
	return minerFundsChange, nil
}

func DecodeMinerDebtChanges(ctx context.Context, store adt.Store, debt cid.Cid) (*minerdiff.DebtChange, error) {
	var minerDebtChange *minerdiff.DebtChange
	if err := store.Get(ctx, debt, minerDebtChange); err != nil {
		return nil, err
	}
	return minerDebtChange, nil
}

func DecodeMinerSectorStatusChanges(ctx context.Context, store adt.Store, sectorStatus cid.Cid) (*minerdiff.SectorStatusChange, error) {
	minerSectorStatusChange := new(minerdiff.SectorStatusChange)
	if err := store.Get(ctx, sectorStatus, minerSectorStatusChange); err != nil {
		return nil, err
	}
	return minerSectorStatusChange, nil
}

func DecodeMinerSectorChanges(ctx context.Context, store adt.Store, sectors cid.Cid) (minerdiff.SectorChangeList, error) {
	sectorArr, err := adt.AsArray(store, sectors, 5)
	if err != nil {
		return nil, err
	}
	sectorChangeList := make(minerdiff.SectorChangeList, 0, sectorArr.Length())
	sectorChange := new(minerdiff.SectorChange)
	if err := sectorArr.ForEach(sectorChange, func(sectorNumber int64) error {
		val := new(minerdiff.SectorChange)
		*val = *sectorChange
		sectorChangeList = append(sectorChangeList, val)
		return nil
	}); err != nil {
		return nil, err
	}
	return sectorChangeList, nil
}

func DecodeMinerPreCommitChanges(ctx context.Context, store adt.Store, precommits cid.Cid) (minerdiff.PreCommitChangeList, error) {
	precommitArr, err := adt.AsArray(store, precommits, 5)
	if err != nil {
		return nil, err
	}
	precommitChangeList := make(minerdiff.PreCommitChangeList, 0, precommitArr.Length())
	precommitChange := new(minerdiff.PreCommitChange)
	if err := precommitArr.ForEach(precommitChange, func(sectorNumber int64) error {
		val := new(minerdiff.PreCommitChange)
		*val = *precommitChange
		precommitChangeList = append(precommitChangeList, val)
		return nil
	}); err != nil {
		return nil, err
	}
	return precommitChangeList, nil
}
