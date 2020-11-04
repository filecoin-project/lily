package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type MinerTaskResult struct {
	Posts map[uint64]cid.Cid

	MinerInfoModel           *MinerInfo
	FeeDebtModel             *MinerFeeDebt
	LockedFundsModel         *MinerLockedFund
	CurrentDeadlineInfoModel *MinerCurrentDeadlineInfo
	PreCommitsModel          MinerPreCommitInfoList
	SectorsModel             MinerSectorInfoList
	SectorEventsModel        MinerSectorEventList
	SectorDealsModel         MinerSectorDealList
}

func (res *MinerTaskResult) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if res.PreCommitsModel != nil {
		if err := res.PreCommitsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.SectorsModel != nil {
		if err := res.SectorsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if len(res.SectorEventsModel) > 0 {
		if err := res.SectorEventsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.MinerInfoModel != nil {
		if err := res.MinerInfoModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.LockedFundsModel != nil {
		if err := res.LockedFundsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.FeeDebtModel != nil {
		if err := res.FeeDebtModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.CurrentDeadlineInfoModel != nil {
		if err := res.CurrentDeadlineInfoModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if res.SectorDealsModel != nil {
		if err := res.SectorDealsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func (res *MinerTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskResult.Persist")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return res.PersistWithTx(ctx, tx)
	})
}

type MinerTaskResultList []*MinerTaskResult

func (ml MinerTaskResultList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, res := range ml {
		if err := res.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
