package tasks

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/lens"
)

// ChangeType denotes type of state change
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

func (c ChangeType) String() string {
	switch c {
	case ChangeTypeUnknown:
		return "unknown"
	case ChangeTypeAdd:
		return "add"
	case ChangeTypeRemove:
		return "remove"
	case ChangeTypeModify:
		return "modify"
	}
	panic("unreachable")
}

type ActorStateChange struct {
	Actor      types.Actor
	ChangeType ChangeType
}

type ActorStateChangeDiff map[address.Address]ActorStateChange

type ActorStatesByType map[string][]*types.ActorV5

type ActorInfo struct {
	Actor       *types.Actor
	ActorName   string
	ActorFamily string
}

type DataSource interface {
	TipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error)
	ActorInfo(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*ActorInfo, error)
	CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error)
	MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error)
	ActorStateChanges(ctx context.Context, ts, pts *types.TipSet) (ActorStateChangeDiff, error)
	MessageExecutions(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error)
	Store() adt.Store

	ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error)
	TipSetBlockMessages(ctx context.Context, ts *types.TipSet) ([]*lens.BlockMessages, error)

	TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error)
	MessageReceiptEvents(ctx context.Context, root cid.Cid) ([]types.Event, error)

	DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error)
	DiffPreCommits(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChanges, error)
	DiffPreCommitsV8(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChangesV8, error)

	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)

	ShouldBurnFn(ctx context.Context, ts *types.TipSet) (lens.ShouldBurnFn, error)

	EthGetBlockByHash(ctx context.Context, blkHash ethtypes.EthHash, fullTxInfo bool) (ethtypes.EthBlock, error)
	EthGetTransactionReceipt(ctx context.Context, txHash ethtypes.EthHash) (*api.EthTxReceipt, error)
	ChainGetMessagesInTipset(ctx context.Context, tsk types.TipSetKey) ([]api.Message, error)
	EthGetTransactionByHash(ctx context.Context, txHash *ethtypes.EthHash) (*ethtypes.EthTx, error)
	StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error)
}
