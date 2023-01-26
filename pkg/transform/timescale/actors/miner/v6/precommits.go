package v6

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v6"

	miner "github.com/filecoin-project/specs-actors/v6/actors/builtin/miner"
)

type PreCommit struct{}

func (PreCommit) Transform(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error) {
	var precommits []*miner.SectorPreCommitOnChainInfo
	for _, change := range change.PreCommitChanges {
		// only care about precommits added
		if change.Change != core.ChangeTypeAdd {
			continue
		}
		precommit := new(miner.SectorPreCommitOnChainInfo)
		if err := precommit.UnmarshalCBOR(bytes.NewReader(change.Current.Raw)); err != nil {
			return nil, err
		}
		precommits = append(precommits, precommit)
	}
	return MinerPreCommitChangesAsModel(ctx, current, addr, precommits)
}

func MinerPreCommitChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, precommits []*miner.SectorPreCommitOnChainInfo) (model.Persistable, error) {
	preCommitModel := make(minermodel.MinerPreCommitInfoList, len(precommits))
	for i, preCommit := range precommits {
		deals := make([]uint64, len(preCommit.Info.DealIDs))
		for didx, deal := range preCommit.Info.DealIDs {
			deals[didx] = uint64(deal)
		}
		preCommitModel[i] = &minermodel.MinerPreCommitInfo{
			Height:                 int64(current.Height()),
			MinerID:                addr.String(),
			SectorID:               uint64(preCommit.Info.SectorNumber),
			StateRoot:              current.ParentState().String(),
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
