package cbor

import (
	"context"
	"fmt"
	"sort"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/extract/procesor"
	"github.com/filecoin-project/lily/pkg/util"
)

type ActorIPLDContainer struct {
	// HAMT of actor addresses to their changes
	// HAMT of miner address to some big structure...
}

type MinerStateChange struct {
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

func ProcessActors(ctx context.Context, store adt.Store, changes *procesor.ActorStateChanges) (interface{}, error) {
	m, err := adt.MakeEmptyMap(store, 5 /*TODO*/)
	if err != nil {
		return nil, err
	}
	if err := HandleMinerChanges(ctx, store, m, changes.MinerActors); err != nil {
		return nil, err
	}
	return nil, nil
}

func HandleMinerChanges(ctx context.Context, store adt.Store, actorHamt *adt.Map, miners map[address.Address]*minerdiff.StateDiff) error {
	for addr, change := range miners {
		msc, err := HandleMinerChange(ctx, store, addr, change)
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

func HandleMinerChange(ctx context.Context, store adt.Store, miner address.Address, change *minerdiff.StateDiff) (*MinerStateChange, error) {
	out := &MinerStateChange{
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
		c, err := MinerInfoHandler(ctx, store, change.InfoChange)
		if err != nil {
			return nil, err
		}
		out.Info = &c
	}
	if change.SectorChanges != nil && len(change.SectorChanges) > 0 {
		c, err := MinerSectorHandler(ctx, store, change.SectorChanges)
		if err != nil {
			return nil, err
		}
		out.Sectors = &c
	}
	if change.PreCommitChanges != nil && len(change.PreCommitChanges) > 0 {
		c, err := MinerPreCommitHandler(ctx, store, change.PreCommitChanges)
		if err != nil {
			return nil, err
		}
		out.PreCommits = &c
	}
	return out, nil
}

func MinerPreCommitHandler(ctx context.Context, store adt.Store, list minerdiff.PreCommitChangeList) (cid.Cid, error) {
	// HACK: this isn't ideal, but we need deterministic ordering and lack a native IPLD ordered set.
	sort.Slice(list, func(i, j int) bool {
		return util.MustCidOf(&list[i].PreCommit).KeyString() < util.MustCidOf(&list[j].PreCommit).KeyString()
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

func MinerSectorHandler(ctx context.Context, store adt.Store, list minerdiff.SectorChangeList) (cid.Cid, error) {
	// HACK: this isn't ideal, but we need deterministic ordering and lack a native IPLD ordered set.
	sort.Slice(list, func(i, j int) bool {
		return util.MustCidOf(&list[i].Sector).KeyString() < util.MustCidOf(&list[j].Sector).KeyString()
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

type MinerInfo struct {
	Info   cid.Cid         `cborgen:"info"`
	Change core.ChangeType `cborgen:"change"`
}

func MinerInfoHandler(ctx context.Context, store adt.Store, info *minerdiff.InfoChange) (cid.Cid, error) {
	// ensure the miner info CID is the same as the CID found on chain.
	infoCid, err := util.CidOf(&info.Info)
	if err != nil {
		return cid.Undef, err
	}
	infoChange := &MinerInfo{
		Info:   infoCid,
		Change: info.Change,
	}
	return store.Put(ctx, infoChange)
}
