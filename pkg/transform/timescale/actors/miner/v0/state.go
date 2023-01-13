package v0

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"

	miner "github.com/filecoin-project/specs-actors/actors/builtin/miner"
)

func ExtractMinerStateChanges(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *actordiff.ActorChange) (model.Persistable, error) {
	if change.Change == core.ChangeTypeRemove {
		return nil, nil
	}
	var out model.PersistableList
	currentState := new(miner.State)
	if err := currentState.UnmarshalCBOR(bytes.NewReader(change.Current)); err != nil {
		return nil, err
	}
	if change.Change == core.ChangeTypeAdd {
		out = append(out, &minermodel.MinerFeeDebt{
			Height:    int64(current.Height()),
			MinerID:   addr.String(),
			StateRoot: current.ParentState().String(),
			FeeDebt:   big.Zero().String(), // v0 has no fee debt
		})
		out = append(out, &minermodel.MinerLockedFund{
			Height:            int64(current.Height()),
			MinerID:           addr.String(),
			StateRoot:         current.ParentState().String(),
			LockedFunds:       currentState.LockedFunds.String(),
			InitialPledge:     currentState.InitialPledgeRequirement.String(),
			PreCommitDeposits: currentState.PreCommitDeposits.String(),
		})
	}
	if change.Change == core.ChangeTypeModify {
		previousState := new(miner.State)
		if err := previousState.UnmarshalCBOR(bytes.NewReader(change.Previous)); err != nil {
			return nil, err
		}
		if !currentState.LockedFunds.Equals(previousState.LockedFunds) ||
			!currentState.InitialPledgeRequirement.Equals(previousState.InitialPledgeRequirement) ||
			!currentState.PreCommitDeposits.Equals(previousState.PreCommitDeposits) {
			out = append(out, &minermodel.MinerLockedFund{
				Height:            int64(current.Height()),
				MinerID:           addr.String(),
				StateRoot:         current.ParentState().String(),
				LockedFunds:       currentState.LockedFunds.String(),
				InitialPledge:     currentState.InitialPledgeRequirement.String(),
				PreCommitDeposits: currentState.PreCommitDeposits.String(),
			})
		}
	}
	return out, nil
}
