package miner

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/util"
)

type StateChange struct {
	// Miner is the address of the miner that changes.
	Miner address.Address `cborgen:"miner"`
	// Funds is the funds that changed for this miner or empty.
	Funds *minerdiff.FundsChange `cborgen:"funds"`
	// Debt is the debt that changed for this miner or empty.
	Debt *minerdiff.DebtChange `cborgen:"debt"`
	// SectorStatus is the sectors whose status changed for this miner or empty.
	SectorStatus *minerdiff.SectorStatusChange `cborgen:"sector_status"`
	// Info is the cid of the miner change info that changed for this miner or empty.
	Info *cid.Cid `cborgen:"info"`
	// PreCommits is an AMT of the pre commits that changed for this miner or empty.
	PreCommits *cid.Cid `cborgen:"pre_commits"`
	// Sectors is an AMT of the sectors that changed for this miner or empty.
	Sectors *cid.Cid `cborgen:"sectors"`
}

func HandleChanges(ctx context.Context, store adt.Store, actorHamt *adt.Map, miners map[address.Address]*minerdiff.StateDiff) error {
	for addr, change := range miners {
		msc, err := ChangeHandler(ctx, store, addr, change)
		if err != nil {
			return err
		}
		if err := actorHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return err
		}
		fmt.Printf("%+v\n", msc)
	}
	return nil
}

func ChangeHandler(ctx context.Context, store adt.Store, miner address.Address, change *minerdiff.StateDiff) (*StateChange, error) {
	out := &StateChange{
		Miner: miner,
	}
	if change.FundsChange != nil {
		out.Funds = change.FundsChange
	}
	if change.DebtChange != nil {
		out.Debt = change.DebtChange
	}
	if change.SectorStatusChanges != nil {
		out.SectorStatus = change.SectorStatusChanges
	}
	if change.InfoChange != nil {
		c, err := InfoHandler(ctx, store, change.InfoChange)
		if err != nil {
			return nil, err
		}
		out.Info = &c
	}
	if change.SectorChanges != nil && len(change.SectorChanges) > 0 {
		c, err := SectorHandler(ctx, store, change.SectorChanges)
		if err != nil {
			return nil, err
		}
		out.Sectors = &c
	}
	if change.PreCommitChanges != nil && len(change.PreCommitChanges) > 0 {
		c, err := PreCommitHandler(ctx, store, change.PreCommitChanges)
		if err != nil {
			return nil, err
		}
		out.PreCommits = &c
	}
	return out, nil
}

func PreCommitHandler(ctx context.Context, store adt.Store, list minerdiff.PreCommitChangeList) (cid.Cid, error) {
	// HACK: this isn't ideal, but we need deterministic ordering and lack a native IPLD ordered set.
	sort.Slice(list, func(i, j int) bool {
		cmp := bytes.Compare(list[i].SectorNumber, list[j].SectorNumber)
		switch cmp {
		case 1:
			return true
		case -1:
			return false
		default:
			panic("precommit with same ID changed twice in one epoch, not possible")
		}
	})
	arr, err := adt.MakeEmptyArray(store, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range list {
		if err := arr.AppendContinuous(l); err != nil {
			return cid.Undef, err
		}
	}
	return arr.Root()
}

func SectorHandler(ctx context.Context, store adt.Store, list minerdiff.SectorChangeList) (cid.Cid, error) {
	// HACK: this isn't ideal, but we need deterministic ordering and lack a native IPLD ordered set.
	sort.Slice(list, func(i, j int) bool {
		return list[i].SectorNumber < list[j].SectorNumber
	})
	arr, err := adt.MakeEmptyArray(store, 5 /*TODO*/)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range list {
		if err := arr.AppendContinuous(l); err != nil {
			return cid.Undef, err
		}
	}
	return arr.Root()
}

type Info struct {
	Info   cid.Cid         `cborgen:"info"`
	Change core.ChangeType `cborgen:"change"`
}

func InfoHandler(ctx context.Context, store adt.Store, info *minerdiff.InfoChange) (cid.Cid, error) {
	// ensure the miner info CID is the same as the CID found on chain.
	infoCid, err := util.CidOf(&info.Info)
	if err != nil {
		return cid.Undef, err
	}
	infoChange := &Info{
		Info:   infoCid,
		Change: info.Change,
	}
	mInfoCid, err := store.Put(ctx, infoCid)
	if err != nil {
		return cid.Undef, err
	}
	if !mInfoCid.Equals(infoCid) {
		panic("here bad")
	}
	return store.Put(ctx, infoChange)
}
