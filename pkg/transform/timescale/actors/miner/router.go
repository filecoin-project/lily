package miner

import (
	"context"
	"fmt"

	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	v1 "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

func TransformMinerState(ctx context.Context, s store.Store, version actortypes.Version, current, executed *types.TipSet, root *cid.Cid) (model.Persistable, error) {
	if root == nil {
		sectorInfoTaskName := tasktype.MinerSectorInfoV7
		if version < actortypes.Version7 {
			sectorInfoTaskName = tasktype.MinerSectorInfoV1_6
		}
		return model.PersistableList{
			data.StartProcessingReport(tasktype.MinerInfo, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.MinerPreCommitInfo, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.MinerSectorDeal, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.MinerSectorEvent, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(sectorInfoTaskName, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
		}, nil
	}
	switch version {
	case actortypes.Version0,
		actortypes.Version2,
		actortypes.Version3,
		actortypes.Version4,
		actortypes.Version5,
		actortypes.Version6,
		actortypes.Version7,
		actortypes.Version8,
		actortypes.Version9:
		return v1.TransformMinerStates(ctx, s, version, current, executed, *root)
	}
	return nil, fmt.Errorf("unsupported version : %d", version)

}
