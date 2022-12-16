package v0

import (
	"bytes"
	"context"
	_ "embed"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*FundsChange)(nil)

type FundsChange struct {
	VestingFunds             abi.TokenAmount `cborgen:"vesting_funds"`
	InitialPledgeRequirement abi.TokenAmount `cborgen:"initial_pledge_requirement"`
	PreCommitDeposit         abi.TokenAmount `cborgen:"pre_commit_deposit"`
	Change                   core.ChangeType `cborgen:"change"`
}

func (f *FundsChange) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := f.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (f *FundsChange) ToStorageBlock() (block.Block, error) {
	data, err := f.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.WithCodec(cid.Raw).Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func DecodeFunds(b []byte) (*FundsChange, error) {
	var funds FundsChange
	if err := funds.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, err
	}

	return &funds, nil
}

const KindMinerFunds = "miner_funds"

func (f *FundsChange) Kind() actors.ActorStateKind {
	return KindMinerFunds
}

var _ actors.ActorStateDiff = (*Funds)(nil)

type Funds struct{}

func (Funds) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerFunds, zap.Inline(act), "duration", time.Since(start))
	}()
	return FundsDiff(ctx, api, act)
}

func FundsDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	// was removed, no change
	if act.Type == core.ChangeTypeRemove {
		// TODO is this correct? Can a miner be removed from the state who still has funds? Would it be better to persist its last known funds value? the modified case below will have persisted that.
		return &FundsChange{
			VestingFunds:             big.Zero(),
			InitialPledgeRequirement: big.Zero(),
			PreCommitDeposit:         big.Zero(),
			Change:                   core.ChangeTypeRemove,
		}, nil
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
			Change:                   core.ChangeTypeAdd,
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
		Change:                   core.ChangeTypeModify,
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
