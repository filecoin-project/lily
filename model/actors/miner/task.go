package miner

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
)

type PartitionStatus struct {
	Terminated bitfield.BitField
	Expired    bitfield.BitField
	Faulted    bitfield.BitField
	InRecovery bitfield.BitField
	Recovered  bitfield.BitField
}

type MinerTaskResult struct {
	Ts        types.TipSetKey
	Pts       types.TipSetKey
	StateRoot cid.Cid

	Addr  address.Address
	Actor *types.Actor

	State            miner.State
	Info             *miner.MinerInfo
	Power            *api.MinerPower
	PreCommitChanges *state.MinerPreCommitChanges
	SectorChanges    *state.MinerSectorChanges
	PartitionDiff    map[uint64]*PartitionStatus
}

func (mtr *MinerTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	tx, err := db.BeginContext(ctx)
	if err != nil {
		return err
	}
	if err := NewMinerStateModel(mtr).PersistWitTx(ctx, tx); err != nil {
		return err
	}
	if err := NewMinerPowerModel(mtr).PersistWithTx(ctx, tx); err != nil {
		return err
	}
	if mtr.PreCommitChanges != nil {
		if err := NewMinerPreCommitInfos(mtr).PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtr.SectorChanges != nil {
		if err := NewMinerSectorInfos(mtr).PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return tx.CommitContext(ctx)
}
