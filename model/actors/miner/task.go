package miner

import (
	"context"

	"github.com/filecoin-project/lily/model"
)

type MinerTaskResult struct {
	Posts MinerSectorPostList

	MinerInfoModel           *MinerInfo
	FeeDebtModel             *MinerFeeDebt
	LockedFundsModel         *MinerLockedFund
	CurrentDeadlineInfoModel *MinerCurrentDeadlineInfo
	PreCommitsModel          MinerPreCommitInfoList
	SectorsModelV1_6         MinerSectorInfoV1_6List
	SectorsModelV7           MinerSectorInfoV7List
	SectorEventsModel        MinerSectorEventList
	SectorDealsModel         MinerSectorDealList
}

func (res *MinerTaskResult) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if res.PreCommitsModel != nil {
		if err := res.PreCommitsModel.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.SectorsModelV1_6 != nil {
		if err := res.SectorsModelV1_6.Persist(ctx, s, version); err != nil {
			return err
		}
	}
	if res.SectorsModelV7 != nil {
		if err := res.SectorsModelV7.Persist(ctx, s, version); err != nil {
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
