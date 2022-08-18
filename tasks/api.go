package tasks

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"

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

type BlockMessageReceipts struct {
	Block   *types.BlockHeader
	Message []*types.Message
	Receipt []*types.MessageReceipt
}

func (bmr *BlockMessageReceipts) Iterator() (*MessageReceiptIterator, error) {
	if len(bmr.Message) != len(bmr.Receipt) {
		return nil, fmt.Errorf("invalid construction, expected equal number receipts (%d) and messages (%d)", len(bmr.Receipt), len(bmr.Message))
	}
	return &MessageReceiptIterator{
		idx:      0,
		msgs:     bmr.Message,
		receipts: bmr.Receipt,
	}, nil
}

type MessageReceiptIterator struct {
	idx      int
	msgs     []*types.Message
	receipts []*types.MessageReceipt
}

func (mri *MessageReceiptIterator) HasNext() bool {
	if mri.idx < len(mri.msgs) {
		return true
	}
	return false
}

func (mri *MessageReceiptIterator) Next() (*types.Message, *types.MessageReceipt) {
	if mri.HasNext() {
		msg := mri.msgs[mri.idx]
		rec := mri.receipts[mri.idx]
		mri.idx++
		return msg, rec
	}
	return nil, nil
}

func (mri *MessageReceiptIterator) Reset() {
	mri.idx = 0
}

func (mri *MessageReceiptIterator) Index() int {
	return mri.idx
}

type DataSource interface {
	TipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error)
	CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error)
	MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error)
	ActorStateChanges(ctx context.Context, ts, pts *types.TipSet) (ActorStateChangeDiff, error)
	MessageExecutions(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error)
	ExecutedAndBlockMessages(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
	Store() adt.Store

	TipSetMessages(ctx context.Context, ts *types.TipSet) ([]types.ChainMsg, error)
	TipSetBlockMessages(ctx context.Context, ts *types.TipSet) ([]*lens.BlockMessages, error)

	TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*BlockMessageReceipts, error)

	DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error)
	DiffPreCommits(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChanges, error)

	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)
}
