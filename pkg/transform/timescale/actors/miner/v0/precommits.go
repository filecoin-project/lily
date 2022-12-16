package v0

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

type PreCommits struct{}

func (PreCommits) Extract(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *v0.StateDiffResult) (model.Persistable, error) {
	var precommits []*miner0.SectorPreCommitOnChainInfo
	for _, change := range change.PreCommitChanges {
		// only care about precommits added
		if change.Change != core.ChangeTypeAdd {
			continue
		}
		if err := core.StateReadDeferred(ctx, change.Current, func(precommit *miner0.SectorPreCommitOnChainInfo) error {
			precommits = append(precommits, precommit)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return MinerPreCommitChangesAsModel(ctx, current, addr, precommits)
}

func MinerPreCommitChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, precommits []*miner0.SectorPreCommitOnChainInfo) (model.Persistable, error) {
	preCommitModel := make(minermodel.MinerPreCommitInfoList, len(precommits))
	for i, preCommit := range precommits {
		preCommitModel[i] = &minermodel.MinerPreCommitInfo{
			Height:                 int64(current.Height()),
			StateRoot:              current.ParentState().String(),
			MinerID:                addr.String(),
			SectorID:               uint64(preCommit.Info.SectorNumber),
			SealedCID:              preCommit.Info.SealedCID.String(),
			SealRandEpoch:          int64(preCommit.Info.SealRandEpoch),
			ExpirationEpoch:        int64(preCommit.Info.Expiration),
			PreCommitDeposit:       preCommit.PreCommitDeposit.String(),
			PreCommitEpoch:         int64(preCommit.PreCommitEpoch),
			DealWeight:             preCommit.DealWeight.String(),
			VerifiedDealWeight:     preCommit.VerifiedDealWeight.String(),
			IsReplaceCapacity:      preCommit.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  preCommit.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: preCommit.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(preCommit.Info.ReplaceSectorNumber),
		}
	}

	return preCommitModel, nil

}
