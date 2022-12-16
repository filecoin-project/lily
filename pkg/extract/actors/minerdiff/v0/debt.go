package v0

import (
	"bytes"
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
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

func (d *DebtChange) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := d.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d *DebtChange) ToStorageBlock() (block.Block, error) {
	data, err := d.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.WithCodec(cid.Raw).Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func DecodeDebt(b []byte) (*DebtChange, error) {
	var debt DebtChange
	if err := debt.UnmarshalCBOR(bytes.NewReader(b)); err != nil {
		return nil, err
	}

	return &debt, nil
}

const KindMinerDebt = "miner_debt"

func (d *DebtChange) Kind() actors.ActorStateKind {
	return KindMinerDebt
}

var _ actors.ActorStateDiff = (*Debt)(nil)

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
