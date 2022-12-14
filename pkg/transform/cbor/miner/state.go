package miner

import (
	"bytes"
	"context"
	"sort"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

var DB *gorm.DB

func init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if err := DB.AutoMigrate(&StateChangeModel{}); err != nil {
		panic(err)
	}
}

type StateChangeModel struct {
	gorm.Model
	Miner        string
	Funds        string
	Debt         string
	SectorStatus string
	Info         string
	PreCommits   string
	Sectors      string
}

func (sc *StateChange) ToStorage(db *gorm.DB) error {
	var (
		funds     = ""
		debt      = ""
		status    = ""
		info      = ""
		precommit = ""
		sector    = ""
	)
	if sc.Funds != nil {
		funds = sc.Funds.String()
	}
	if sc.Debt != nil {
		debt = sc.Debt.String()
	}
	if sc.SectorStatus != nil {
		status = sc.SectorStatus.String()
	}
	if sc.Info != nil {
		info = sc.Info.String()
	}
	if sc.PreCommits != nil {
		precommit = sc.PreCommits.String()
	}
	if sc.Sectors != nil {
		sector = sc.Sectors.String()
	}
	tx := db.Create(&StateChangeModel{
		Miner:        sc.Miner.String(),
		Funds:        funds,
		Debt:         debt,
		SectorStatus: status,
		Info:         info,
		PreCommits:   precommit,
		Sectors:      sector,
	})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

type StateChange struct {
	// Miner is the address of the miner that changes.
	Miner address.Address `cborgen:"miner"`
	// Funds is the funds that changed for this miner or empty.
	Funds *cid.Cid `cborgen:"funds"`
	// Debt is the debt that changed for this miner or empty.
	Debt *cid.Cid `cborgen:"debt"`
	// SectorStatus is the sectors whose status changed for this miner or empty.
	SectorStatus *cid.Cid `cborgen:"sector_status"`
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
		if msc == nil {
			continue
		}
		if err := msc.ToStorage(DB); err != nil {
			return err
		}
		if err := actorHamt.Put(abi.AddrKey(addr), msc); err != nil {
			return err
		}
	}
	return nil
}

func ChangeHandler(ctx context.Context, store adt.Store, miner address.Address, change *minerdiff.StateDiff) (*StateChange, error) {
	hasChangedState := false
	out := &StateChange{}
	if change.FundsChange != nil {
		hasChangedState = true
		c, err := FundsHandler(ctx, store, change.FundsChange)
		if err != nil {
			return nil, err
		}
		out.Funds = &c
	}
	if change.DebtChange != nil {
		hasChangedState = true
		c, err := DebtHandler(ctx, store, change.DebtChange)
		if err != nil {
			return nil, err
		}
		out.Debt = &c
	}
	if change.SectorStatusChanges != nil {
		hasChangedState = true
		c, err := SectorStatusHandler(ctx, store, change.SectorStatusChanges)
		if err != nil {
			return nil, err
		}
		out.SectorStatus = &c
	}
	if change.InfoChange != nil {
		hasChangedState = true
		c, err := InfoHandler(ctx, store, change.InfoChange)
		if err != nil {
			return nil, err
		}
		out.Info = &c
	}
	if change.SectorChanges != nil && len(change.SectorChanges) > 0 {
		hasChangedState = true
		c, err := SectorHandler(ctx, store, change.SectorChanges)
		if err != nil {
			return nil, err
		}
		out.Sectors = &c
	}
	if change.PreCommitChanges != nil && len(change.PreCommitChanges) > 0 {
		hasChangedState = true
		c, err := PreCommitHandler(ctx, store, change.PreCommitChanges)
		if err != nil {
			return nil, err
		}
		out.PreCommits = &c
	}
	if hasChangedState {
		out.Miner = miner
		return out, nil
	}
	return nil, nil
}

func FundsHandler(ctx context.Context, store adt.Store, funds *minerdiff.FundsChange) (cid.Cid, error) {
	return store.Put(ctx, funds)
}

func DebtHandler(ctx context.Context, store adt.Store, debt *minerdiff.DebtChange) (cid.Cid, error) {
	return store.Put(ctx, debt)
}

func SectorStatusHandler(ctx context.Context, store adt.Store, sectors *minerdiff.SectorStatusChange) (cid.Cid, error) {
	return store.Put(ctx, sectors)
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
	mInfoCid, err := store.Put(ctx, info.Info)
	if err != nil {
		return cid.Undef, err
	}
	infoChange := &Info{
		Info:   mInfoCid,
		Change: info.Change,
	}
	return store.Put(ctx, infoChange)
}
