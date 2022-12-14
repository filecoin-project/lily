package minerdiff

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*DebtChange)(nil)

type DebtChange struct {
	FeeDebt abi.TokenAmount `cborgen:"fee_debt"`
	Change  core.ChangeType `cborgen:"change"`
}

const KindMinerDebt = "miner_debt"

func (d *DebtChange) Kind() actors.ActorStateKind {
	return KindMinerDebt
}

var _ actors.ActorDiffer = (*Debt)(nil)

type Debt struct{}

func (Debt) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerDebt, zap.Inline(act), "duration", time.Since(start))
	}()
	return DebtDiff(ctx, api, act)
}

func DebtDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	// was removed, its debt is gone...
	// TODO is this correct? Can a miner be removed from the state who has debt? Would it be better to persist its last known debt value? the modified case below will have persisted that.
	if act.Type == core.ChangeTypeRemove {
		return &DebtChange{
			FeeDebt: big.Zero(),
			Change:  core.ChangeTypeRemove,
		}, nil
	}
	currentMiner, err := api.MinerLoad(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}
	currentDebt, err := currentMiner.FeeDebt()
	if err != nil {
		return nil, err
	}
	// added, all debt (assumed to be zero) is new
	if act.Type == core.ChangeTypeAdd {
		return &DebtChange{FeeDebt: currentDebt, Change: core.ChangeTypeAdd}, nil
	}
	// actor state was modified, check if debt differ.

	executedMiner, err := api.MinerLoad(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}
	executedDebt, err := executedMiner.FeeDebt()
	if err != nil {
		return nil, err
	}
	// no change if equal.
	if executedDebt.Equals(currentDebt) {
		return nil, nil
	}
	return &DebtChange{FeeDebt: currentDebt, Change: core.ChangeTypeModify}, nil
}
