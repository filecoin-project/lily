package v0

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v0 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

func HandleMinerFundsChange(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, changes *v0.FundsChange) (model.Persistable, error) {
	return &minermodel.MinerLockedFund{
		Height:            int64(current.Height()),
		MinerID:           addr.String(),
		StateRoot:         current.ParentState().String(),
		LockedFunds:       changes.VestingFunds.String(),
		InitialPledge:     changes.InitialPledgeRequirement.String(),
		PreCommitDeposits: changes.PreCommitDeposit.String(),
	}, nil
}
