package minerdiff

import (
	"context"
	_ "embed"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*FundsChange)(nil)

type FundsChange struct {
	VestingFunds             abi.TokenAmount
	InitialPledgeRequirement abi.TokenAmount
	PreCommitDeposit         abi.TokenAmount
}

const KindMinerFunds = "miner_funds"

func (f *FundsChange) Kind() actors.ActorStateKind {
	return KindMinerFunds
}

var _ actors.ActorDiffer = (*Funds)(nil)

type Funds struct{}

func (Funds) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	return FundsDiff(ctx, api, act)
}

func FundsDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	// was removed, no change
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}
	currentMiner, err := api.MinerLoad(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}
	currentFunds, err := currentMiner.LockedFunds()
	if err != nil {
		return nil, err
	}
	// added, all funds are new.
	if act.Type == core.ChangeTypeAdd {
		return &FundsChange{
			VestingFunds:             currentFunds.VestingFunds,
			InitialPledgeRequirement: currentFunds.InitialPledgeRequirement,
			PreCommitDeposit:         currentFunds.PreCommitDeposits,
		}, nil
	}

	executedMiner, err := api.MinerLoad(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}
	executedFunds, err := executedMiner.LockedFunds()
	if err != nil {
		return nil, err
	}
	// no change if equal
	if LockedFundsEqual(currentFunds, executedFunds) {
		return nil, nil
	}
	// funds differ
	return &FundsChange{
		VestingFunds:             currentFunds.VestingFunds,
		InitialPledgeRequirement: currentFunds.InitialPledgeRequirement,
		PreCommitDeposit:         currentFunds.PreCommitDeposits,
	}, nil
}

func LockedFundsEqual(cur, pre miner.LockedFunds) bool {
	if !cur.VestingFunds.Equals(pre.VestingFunds) {
		return true
	}
	if !cur.PreCommitDeposits.Equals(pre.PreCommitDeposits) {
		return true
	}
	if !cur.InitialPledgeRequirement.Equals(pre.InitialPledgeRequirement) {
		return true
	}
	return false
}
