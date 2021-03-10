package lens

import (
	"context"

	"github.com/filecoin-project/lotus/node/modules/dtypes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
)

type API interface {
	StoreAPI
	ChainAPI
	StateAPI

	GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*ExecutedMessage, error)
}

type StoreAPI interface {
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

	ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error)
	ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]api.Message, error)
	ChainGetParentReceipts(ctx context.Context, blockCid cid.Cid) ([]*types.MessageReceipt, error)
}

type StateAPI interface {
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateListActors(context.Context, types.TipSetKey) ([]address.Address, error)
	StateChangedActors(context.Context, cid.Cid, cid.Cid) (map[string]types.Actor, error)

	StateMinerSectors(ctx context.Context, addr address.Address, bf *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)

	StateMarketDeals(context.Context, types.TipSetKey) (map[string]api.MarketDeal, error)

	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error)
	StateVMCirculatingSupplyInternal(context.Context, types.TipSetKey) (api.CirculatingSupply, error)
	StateNetworkName(context.Context) (dtypes.NetworkName, error)
}

type APICloser func()

type APIOpener interface {
	Open(context.Context) (API, APICloser, error)
}

type ExecutedMessage struct {
	Cid           cid.Cid
	Height        abi.ChainEpoch
	Message       *types.Message
	Receipt       *types.MessageReceipt
	BlockHeader   *types.BlockHeader
	Blocks        []cid.Cid // blocks this message appeared in
	Index         uint64    // Message and receipt sequence in tipset
	FromActorCode cid.Cid   // code of the actor the message is from
	ToActorCode   cid.Cid   // code of the actor the message is to
	GasOutputs    vm.GasOutputs
}
