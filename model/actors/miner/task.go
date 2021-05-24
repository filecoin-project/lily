package miner

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
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

func (res *MinerTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if res.PreCommitsModel != nil {
		if err := res.PreCommitsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.SectorsModel != nil {
		if err := res.SectorsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if len(res.SectorEventsModel) > 0 {
		if err := res.SectorEventsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.MinerInfoModel != nil {
		if err := res.MinerInfoModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.LockedFundsModel != nil {
		if err := res.LockedFundsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.FeeDebtModel != nil {
		if err := res.FeeDebtModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.CurrentDeadlineInfoModel != nil {
		if err := res.CurrentDeadlineInfoModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.SectorDealsModel != nil {
		if err := res.SectorDealsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.Posts != nil {
		if err := res.Posts.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}

type MinerTaskResultList []*MinerTaskResult

func (ml MinerTaskResultList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerTaskResultList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	for _, res := range ml {
		if err := res.Persist(ctx, s, version); err != nil {
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

// Persist PersistModel with every field of MinerTasklists
func (mtl *MinerTaskLists) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if mtl.PreCommitsModel != nil {
		if err := mtl.PreCommitsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.SectorsModel != nil {
		if err := mtl.SectorsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if len(mtl.SectorEventsModel) > 0 {
		if err := mtl.SectorEventsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.MinerInfoModel != nil {
		if err := mtl.MinerInfoModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.LockedFundsModel != nil {
		if err := mtl.LockedFundsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.FeeDebtModel != nil {
		if err := mtl.FeeDebtModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.CurrentDeadlineInfoModel != nil {
		if err := mtl.CurrentDeadlineInfoModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.SectorDealsModel != nil {
		if err := mtl.SectorDealsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if mtl.SectorPostModel != nil {
		if err := mtl.SectorPostModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	return nil
}
