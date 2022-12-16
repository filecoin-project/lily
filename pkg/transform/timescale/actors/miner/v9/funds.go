package v9

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v9 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v9"
)

type Fund struct{}

func (Fund) Extract(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *v9.StateDiffResult) (model.Persistable, error) {
	if change.FundsChange == nil {
		return nil, nil
	}
	funds := change.FundsChange
	return &minermodel.MinerLockedFund{
		Height:            int64(current.Height()),
		MinerID:           addr.String(),
		StateRoot:         current.ParentState().String(),
		LockedFunds:       funds.VestingFunds.String(),
		InitialPledge:     funds.InitialPledgeRequirement.String(),
		PreCommitDeposits: funds.PreCommitDeposit.String(),
	}, nil
}
