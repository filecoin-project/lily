package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type MinerTaskResult struct {
	Posts MinerSectorPostList

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
	if res.Posts != nil {
		if err := res.Posts.PersistWithTx(ctx, tx); err != nil {
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
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskResultList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	for _, res := range ml {
		if err := res.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

// MinerTaskLists allow better batched insertion of Miner-related models.
type MinerTaskLists struct {
	MinerInfoModel           MinerInfoList
	FeeDebtModel             MinerFeeDebtList
	LockedFundsModel         MinerLockedFundsList
	CurrentDeadlineInfoModel MinerCurrentDeadlineInfoList
	PreCommitsModel          MinerPreCommitInfoList
	SectorsModel             MinerSectorInfoList
	SectorEventsModel        MinerSectorEventList
	SectorDealsModel         MinerSectorDealList
	SectorPostModel          MinerSectorPostList
}

// PersistWithTx calls PersistWithTx on every field of MinerTasklists which
// should result in batched commits of the items since they are lists.
func (mtl *MinerTaskLists) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskLists.PersistWithTx")
	defer span.End()

	if mtl.PreCommitsModel != nil {
		if err := mtl.PreCommitsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.SectorsModel != nil {
		if err := mtl.SectorsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if len(mtl.SectorEventsModel) > 0 {
		if err := mtl.SectorEventsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.MinerInfoModel != nil {
		if err := mtl.MinerInfoModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.LockedFundsModel != nil {
		if err := mtl.LockedFundsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.FeeDebtModel != nil {
		if err := mtl.FeeDebtModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.CurrentDeadlineInfoModel != nil {
		if err := mtl.CurrentDeadlineInfoModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.SectorDealsModel != nil {
		if err := mtl.SectorDealsModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	if mtl.SectorPostModel != nil {
		if err := mtl.SectorPostModel.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
