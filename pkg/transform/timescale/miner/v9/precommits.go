package v9

import (
	"context"

	"github.com/filecoin-project/go-address"
	miner9 "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
)

func HandleMinerPreCommitChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, changes minerdiff.PreCommitChangeList) (model.Persistable, error) {
	var precommits []*miner9.SectorPreCommitOnChainInfo
	for _, change := range changes {
		// only care about precommits added
		if change.Change != core.ChangeTypeAdd {
			continue
		}
		if err := core.StateReadDeferred(ctx, change.Current, func(precommit *miner9.SectorPreCommitOnChainInfo) error {
			precommits = append(precommits, precommit)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return MinerPreCommitChangesAsModel(ctx, current, addr, precommits)
}

func MinerPreCommitChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, precommits []*miner9.SectorPreCommitOnChainInfo) (model.Persistable, error) {
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

	return preCommitModel, nil

}
