package miner

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

// PartitionStatus contains bitfileds of sectorID's that are removed, faulted, recovered and recovering.
type PartitionStatus struct {
	Removed    bitfield.BitField
	Faulted    bitfield.BitField
	Recovering bitfield.BitField
	Recovered  bitfield.BitField
}

type MinerTaskResult struct {
	Ts        types.TipSetKey
	Pts       types.TipSetKey
	Height    abi.ChainEpoch
	StateRoot cid.Cid

	Addr  address.Address
	Actor *types.Actor

	State            miner.State
	Info             miner.MinerInfo
	Power            *api.MinerPower
	PreCommitChanges *miner.PreCommitChanges
	SectorChanges    *miner.SectorChanges
	Posts            map[uint64]cid.Cid
	SectorEvents     MinerSectorEventList
}

func (mtr *MinerTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := NewMinerStateModel(mtr).PersistWithTx(ctx, tx); err != nil {
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
		if len(mtr.SectorEvents) > 0 {
			if err := mtr.SectorEvents.PersistWithTx(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
}
