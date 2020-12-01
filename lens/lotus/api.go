package lotus

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
)

func NewAPIWrapper(node api.FullNode, store adt.Store) *APIWrapper {
	return &APIWrapper{
		FullNode: node,
		store:    store,
	}
}

var _ lens.API = &APIWrapper{}

type APIWrapper struct {
	api.FullNode
	store adt.Store
}

func (aw *APIWrapper) Store() adt.Store {
	return aw.store
}

func (aw *APIWrapper) ChainGetBlock(ctx context.Context, msg cid.Cid) (*types.BlockHeader, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetBlock")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetBlock"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetBlock(ctx, msg)
}

func (aw *APIWrapper) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetBlockMessages")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetBlockMessages"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetBlockMessages(ctx, msg)
}

func (aw *APIWrapper) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainNotify")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetGenesis"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetGenesis(ctx)
}

func (aw *APIWrapper) ChainGetParentMessages(ctx context.Context, bcid cid.Cid) ([]api.Message, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetParentMessages")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetParentMessages"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetParentMessages(ctx, bcid)
}

func (aw *APIWrapper) StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateGetReceipt")
	defer span.End()
	return aw.FullNode.StateGetReceipt(ctx, bcid, tsk)
}

func (aw *APIWrapper) ChainGetParentReceipts(ctx context.Context, bcid cid.Cid) ([]*types.MessageReceipt, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetParentReceipts")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetParentReceipts"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetParentReceipts(ctx, bcid)
}

func (aw *APIWrapper) ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetTipSet")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainGetTipSet"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainGetTipSet(ctx, tsk)
}

func (aw *APIWrapper) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainNotify")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainNotify"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainNotify(ctx)
}

func (aw *APIWrapper) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainReadObj")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "ChainReadObj"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.ChainReadObj(ctx, obj)
}

func (aw *APIWrapper) StateChangedActors(ctx context.Context, old cid.Cid, new cid.Cid) (map[string]types.Actor, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateChangedActors")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateChangedActors"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateChangedActors(ctx, old, new)
}

func (aw *APIWrapper) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateGetActor")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateGetActor"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateGetActor(ctx, actor, tsk)
}

func (aw *APIWrapper) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateListActors")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateListActors"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateListActors(ctx, tsk)
}

func (aw *APIWrapper) StateMarketDeals(ctx context.Context, tsk types.TipSetKey) (map[string]api.MarketDeal, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMarketDeals")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateMarketDeals"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateMarketDeals(ctx, tsk)
}

func (aw *APIWrapper) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMinerPower")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateMinerPower"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateMinerPower(ctx, addr, tsk)
}

func (aw *APIWrapper) StateMinerSectors(ctx context.Context, addr address.Address, filter *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMinerSectors")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateMinerSectors"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateMinerSectors(ctx, addr, filter, tsk)
}

func (aw *APIWrapper) StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateReadState")
	defer span.End()
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.API, "StateReadState"))
	stop := metrics.Timer(ctx, metrics.LensRequestDuration)
	defer stop()
	return aw.FullNode.StateReadState(ctx, actor, tsk)
}

func (aw *APIWrapper) ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs {
	return vm.ComputeGasOutputs(gasUsed, gasLimit, baseFee, feeCap, gasPremium)
}

func (aw *APIWrapper) StateVMCirculatingSupplyInternal(ctx context.Context, tsk types.TipSetKey) (api.CirculatingSupply, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateCirculatingSupply")
	defer span.End()
	return aw.FullNode.StateVMCirculatingSupplyInternal(ctx, tsk)
}

// GetExecutedMessagesForTipset returns a list of messages sent as part of pts (parent) with receipts found in ts (child).
// No attempt at deduplication of messages is made.
func (aw *APIWrapper) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	if !types.CidArrsEqual(ts.Parents().Cids(), pts.Cids()) {
		return nil, xerrors.Errorf("child tipset (%s) is not on the same chain as parent (%s)", ts.Key(), pts.Key())
	}

	stateTree, err := state.LoadStateTree(aw.Store(), ts.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}

	// Build a lookup of actor codes
	actorCodes := map[address.Address]cid.Cid{}
	if err := stateTree.ForEach(func(a address.Address, act *types.Actor) error {
		actorCodes[a] = act.Code
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("iterate actors: %w", err)
	}

	getActorCode := func(a address.Address) cid.Cid {
		c, ok := actorCodes[a]
		if ok {
			return c
		}

		return cid.Undef
	}

	// Build a lookup of which block headers indexed by their cid
	blockHeaders := map[cid.Cid]*types.BlockHeader{}
	for _, bh := range pts.Blocks() {
		blockHeaders[bh.Cid()] = bh
	}

	// Build a lookup of which blocks each message appears in
	messageBlocks := map[cid.Cid][]cid.Cid{}

	for _, blkCid := range pts.Cids() {
		blkMsgs, err := aw.ChainGetBlockMessages(ctx, blkCid)
		if err != nil {
			return nil, xerrors.Errorf("get block messages: %w", err)
		}

		for _, mcid := range blkMsgs.Cids {
			messageBlocks[mcid] = append(messageBlocks[mcid], blkCid)
		}
	}

	// Get messages that were processed in the parent tipset
	msgs, err := aw.ChainGetParentMessages(ctx, ts.Cids()[0])
	if err != nil {
		return nil, xerrors.Errorf("get parent messages: %w", err)
	}

	// Get receipts for parent messages
	rcpts, err := aw.ChainGetParentReceipts(ctx, ts.Cids()[0])
	if err != nil {
		return nil, xerrors.Errorf("get parent receipts: %w", err)
	}

	if len(rcpts) != len(msgs) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of receipts: got %d wanted %d", len(rcpts), len(msgs))
	}

	// Start building a list of completed message with receipt
	emsgs := make([]*lens.ExecutedMessage, 0, len(msgs))

	for index, m := range msgs {
		emsgs = append(emsgs, &lens.ExecutedMessage{
			Cid:           m.Cid,
			Height:        pts.Height(),
			Message:       m.Message,
			Receipt:       rcpts[index],
			BlockHeader:   blockHeaders[messageBlocks[m.Cid][0]],
			Blocks:        messageBlocks[m.Cid],
			Index:         uint64(index),
			FromActorCode: getActorCode(m.Message.From),
			ToActorCode:   getActorCode(m.Message.To),
		})
	}

	return emsgs, nil
}
