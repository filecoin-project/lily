package lens

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
)

type API interface {
	StoreAPI
	ChainAPI
	StateAPI
	VMAPI

	GetMessageExecutionsForTipSet(ctx context.Context, ts, pts *types.TipSet) ([]*MessageExecution, error)
	GetMessageExecutionsForTipSetV2(ctx context.Context, ts, pts *types.TipSet) ([]*MessageExecutionV2, error)
}
type StoreAPI interface {
	// TODO this should be the lotus store not the specs-actors store.
	Store() adt.Store
}

type ChainAPI interface {
	ChainNotify(context.Context) (<-chan []*api.HeadChange, error)
	ChainHead(context.Context) (*types.TipSet, error)

	ChainHasObj(ctx context.Context, obj cid.Cid) (bool, error)
	ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error)

	ChainGetGenesis(ctx context.Context) (*types.TipSet, error)
	ChainGetTipSet(context.Context, types.TipSetKey) (*types.TipSet, error)
	ChainGetTipSetByHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)
	ChainGetTipSetAfterHeight(context.Context, abi.ChainEpoch, types.TipSetKey) (*types.TipSet, error)

	ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error)
	ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]api.Message, error)

	ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error)

	MessagesForTipSetBlocks(ctx context.Context, ts *types.TipSet) ([]*BlockMessages, error)
	TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*BlockMessageReceipts, error)
}

type StateAPI interface {
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateListActors(context.Context, types.TipSetKey) ([]address.Address, error)
	StateChangedActors(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)

	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)

	StateMarketDeals(context.Context, types.TipSetKey) (map[string]*api.MarketDeal, error)

	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error)
	CirculatingSupply(context.Context, types.TipSetKey) (api.CirculatingSupply, error)
	StateNetworkName(context.Context) (dtypes.NetworkName, error)
	StateNetworkVersion(ctx context.Context, key types.TipSetKey) (network.Version, error)
}

type ShouldBurnFn func(ctx context.Context, msg *types.Message, errcode exitcode.ExitCode) (bool, error)

type VMAPI interface {
	BurnFundsFn(ctx context.Context, ts *types.TipSet) (ShouldBurnFn, error)
}

type MessageExecution struct {
	Cid       cid.Cid
	StateRoot cid.Cid
	Height    abi.ChainEpoch

	Message *types.Message
	Ret     *vm.ApplyRet

	FromActorCode cid.Cid // code of the actor the message is from
	ToActorCode   cid.Cid // code of the actor the message is to

	Implicit bool
}

type MessageExecutionV2 struct {
	Cid       cid.Cid
	StateRoot cid.Cid
	Height    abi.ChainEpoch

	Message *types.Message
	Ret     *vm.ApplyRet

	Implicit bool
}

type BlockMessages struct {
	Block        *types.BlockHeader     // block messages appeared in
	BlsMessages  []*types.Message       // BLS messages in block `Block`
	SecpMessages []*types.SignedMessage // SECP messages in block `Block`
}

// BlockMessageReceipts contains a block its messages and their corresponding receipts.
// The Receipts are one-to-one with Messages index.
type BlockMessageReceipts struct {
	Block *types.BlockHeader
	// Messages contained in Block.
	Messages []types.ChainMsg
	// Receipts contained in Block.
	Receipts []*types.MessageReceipt
	// MessageExectionIndex contains a mapping of Messages to their execution order in the tipset they were included.
	MessageExecutionIndex map[types.ChainMsg]int
}

type MessageReceiptIterator struct {
	idx      int
	msgs     []types.ChainMsg
	receipts []*types.MessageReceipt
	exeIdx   map[types.ChainMsg]int
}

// Iterator returns a MessageReceiptIterator to conveniently iterate messages, their execution index, and their respective receipts.
func (bmr *BlockMessageReceipts) Iterator() (*MessageReceiptIterator, error) {
	if len(bmr.Messages) != len(bmr.Receipts) {
		return nil, fmt.Errorf("invalid construction, expected equal number receipts (%d) and messages (%d)", len(bmr.Receipts), len(bmr.Messages))
	}
	return &MessageReceiptIterator{
		idx:      0,
		msgs:     bmr.Messages,
		receipts: bmr.Receipts,
		exeIdx:   bmr.MessageExecutionIndex,
	}, nil
}

// HasNext returns `true` while there are messages/receipts to iterate.
func (mri *MessageReceiptIterator) HasNext() bool {
	if mri.idx < len(mri.msgs) {
		return true
	}
	return false
}

// Next returns the next message, its execution index, and receipt in the MessageReceiptIterator.
func (mri *MessageReceiptIterator) Next() (types.ChainMsg, int, *types.MessageReceipt) {
	if mri.HasNext() {
		msg := mri.msgs[mri.idx]
		exeIdx := mri.exeIdx[msg]
		rec := mri.receipts[mri.idx]
		mri.idx++
		return msg, exeIdx, rec
	}
	return nil, -1, nil
}

// Reset resets the MessageReceiptIterator to the first message/receipt.
func (mri *MessageReceiptIterator) Reset() {
	mri.idx = 0
}
