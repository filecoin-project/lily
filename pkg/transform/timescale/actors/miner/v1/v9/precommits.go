package v9

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"

	miner "github.com/filecoin-project/go-state-types/builtin/v9/miner"
)

type PreCommit struct{}

func (PreCommit) Transform(ctx context.Context, current, executed *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.MinerPreCommitInfo, current)
	for _, m := range miners {
		var precommits []*miner.SectorPreCommitOnChainInfo
		for _, change := range m.StateChange.PreCommitChanges {
			// only care about precommits added
			if change.Change != core.ChangeTypeAdd {
				continue
			}
			precommit := new(miner.SectorPreCommitOnChainInfo)
			if err := precommit.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
				report.AddError(err)
			}
			precommits = append(precommits, precommit)
		}
		report.AddModels(MinerPreCommitChangesAsModel(ctx, current, m.Address, precommits))
	}
	return report.Finish()
}

func MinerPreCommitChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, precommits []*miner.SectorPreCommitOnChainInfo) model.Persistable {
	preCommitModel := make(minermodel.MinerPreCommitInfoV9List, len(precommits))
	for i, preCommit := range precommits {
		deals := make([]uint64, len(preCommit.Info.DealIDs))
		for didx, deal := range preCommit.Info.DealIDs {
			deals[didx] = uint64(deal)
		}
		unSealedCID := ""
		if preCommit.Info.UnsealedCid != nil {
			unSealedCID = preCommit.Info.UnsealedCid.String()
		}
		preCommitModel[i] = &minermodel.MinerPreCommitInfoV9{
			Height:           int64(current.Height()),
			StateRoot:        current.ParentState().String(),
			MinerID:          addr.String(),
			SectorID:         uint64(preCommit.Info.SectorNumber),
			PreCommitDeposit: preCommit.PreCommitDeposit.String(),
			PreCommitEpoch:   int64(preCommit.PreCommitEpoch),
			SealedCID:        preCommit.Info.SealedCID.String(),
			SealRandEpoch:    int64(preCommit.Info.SealRandEpoch),
			ExpirationEpoch:  int64(preCommit.Info.Expiration),
			DealIDS:          deals,
			UnsealedCID:      unSealedCID,
		}
	}

	return preCommitModel

}
