package camera

import (
	"context"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	blocks "github.com/ipfs/go-block-format"
)

//
// Store

var _ adt.Store = (*StoreRecorder)(nil)

func NewStoreRecorder(store adt.Store, f *Film) *StoreRecorder {
	return &StoreRecorder{
		store: store,
		film:  f,
	}
}

// an implementation of the adt store recording all CID's read from it.
type StoreRecorder struct {
	store adt.Store

	film *Film
}

func (s *StoreRecorder) Context() context.Context {
	return s.store.Context()
}

func (s *StoreRecorder) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	s.film.Capture(c)
	return s.store.Get(ctx, c, out)
}

func (s *StoreRecorder) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return s.store.Put(ctx, v)
}

//
// API

var _ lens.API = (*APIRecorder)(nil)

func NewAPIRecorder(node api.FullNode, store adt.Store, f *Film) *APIRecorder {
	return &APIRecorder{
		api:   node,
		store: NewStoreRecorder(store, f),
		film:  f,
	}
}

type APIRecorder struct {
	api   api.FullNode
	store *StoreRecorder
	film  *Film
}

func (ar *APIRecorder) Store() adt.Store {
	return ar.store
}

func (ar *APIRecorder) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	return ar.api.ChainNotify(ctx)
}

func (ar *APIRecorder) ChainHead(ctx context.Context) (*types.TipSet, error) {
	head, err := ar.api.ChainHead(ctx)
	if err != nil {
		return nil, err
	}
	ar.film.Capture(head.Cids()...)
	return head, nil
}

func (ar *APIRecorder) ChainHasObj(ctx context.Context, obj cid.Cid) (bool, error) {
	ar.film.Capture(obj)
	return ar.api.ChainHasObj(ctx, obj)
}

func (ar *APIRecorder) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	// record
	ar.film.Capture(obj)
	return ar.api.ChainReadObj(ctx, obj)
}

func (ar *APIRecorder) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	genesis, err := ar.api.ChainGetGenesis(ctx)
	if err != nil {
		return nil, err
	}
	ar.film.Capture(genesis.Cids()...)
	return genesis, nil
}

func (ar *APIRecorder) ChainGetTipSet(ctx context.Context, key types.TipSetKey) (*types.TipSet, error) {
	ts, err := ar.api.ChainGetTipSet(ctx, key)
	if err != nil {
		return nil, err
	}
	ar.film.Capture(ts.Cids()...)
	return ts, nil
}

func (ar *APIRecorder) ChainGetTipSetByHeight(ctx context.Context, epoch abi.ChainEpoch, key types.TipSetKey) (*types.TipSet, error) {
	ts, err := ar.api.ChainGetTipSetByHeight(ctx, epoch, key)
	if err != nil {
		return nil, err
	}
	ar.film.Capture(ts.Cids()...)
	return ts, nil
}

func (ar *APIRecorder) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	ar.film.Capture(msg)
	return ar.api.ChainGetBlockMessages(ctx, msg)
}

func (ar *APIRecorder) ChainGetParentMessages(ctx context.Context, blockCid cid.Cid) ([]api.Message, error) {
	ar.film.Capture(blockCid)
	return ar.api.ChainGetParentMessages(ctx, blockCid)
}

func (ar *APIRecorder) ChainGetParentReceipts(ctx context.Context, blockCid cid.Cid) ([]*types.MessageReceipt, error) {
	ar.film.Capture(blockCid)
	return ar.api.ChainGetParentReceipts(ctx, blockCid)
}

func (ar *APIRecorder) StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	ar.film.Capture(tsk.Cids()...)
	act, err := ar.api.StateGetActor(ctx, addr, tsk)
	if err != nil {
		return nil, err
	}
	ar.film.Capture(act.Head)
	return act, nil
}

func (ar *APIRecorder) StateListActors(ctx context.Context, key types.TipSetKey) ([]address.Address, error) {
	ar.film.Capture(key.Cids()...)
	return ar.api.StateListActors(ctx, key)
}

func (ar *APIRecorder) StateChangedActors(ctx context.Context, c cid.Cid, c2 cid.Cid) (map[string]types.Actor, error) {
	ar.film.Capture(c, c2)
	return ar.api.StateChangedActors(ctx, c, c2)
}

func (ar *APIRecorder) StateMinerSectors(ctx context.Context, addr address.Address, bf *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	ar.film.Capture(tsk.Cids()...)
	return ar.api.StateMinerSectors(ctx, addr, bf, tsk)
}

func (ar *APIRecorder) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	ar.film.Capture(tsk.Cids()...)
	return ar.api.StateMinerPower(ctx, addr, tsk)
}

func (ar *APIRecorder) StateMarketDeals(ctx context.Context, key types.TipSetKey) (map[string]api.MarketDeal, error) {
	ar.film.Capture(key.Cids()...)
	return ar.api.StateMarketDeals(ctx, key)
}

func (ar *APIRecorder) StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	// TODO assume the actor head CID has already been requested I guess?
	ar.film.Capture(tsk.Cids()...)
	return ar.api.StateReadState(ctx, addr, tsk)
}

func (ar *APIRecorder) StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	ar.film.Capture(tsk.Cids()...)
	ar.film.Capture(bcid)
	return ar.api.StateGetReceipt(ctx, bcid, tsk)
}

func (ar *APIRecorder) StateVMCirculatingSupplyInternal(ctx context.Context, key types.TipSetKey) (api.CirculatingSupply, error) {
	ar.film.Capture(key.Cids()...)
	return ar.api.StateVMCirculatingSupplyInternal(ctx, key)
}

func (ar *APIRecorder) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	if !types.CidArrsEqual(ts.Parents().Cids(), pts.Cids()) {
		return nil, xerrors.Errorf("child tipset (%s) is not on the same chain as parent (%s)", ts.Key(), pts.Key())
	}

	stateTree, err := state.LoadStateTree(ar.Store(), ts.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load state tree: %w", err)
	}

	parentStateTree, err := state.LoadStateTree(ar.Store(), pts.ParentState())
	if err != nil {
		return nil, xerrors.Errorf("load parent state tree: %w", err)
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
		blkMsgs, err := ar.ChainGetBlockMessages(ctx, blkCid)
		if err != nil {
			return nil, xerrors.Errorf("get block messages: %w", err)
		}

		for _, mcid := range blkMsgs.Cids {
			messageBlocks[mcid] = append(messageBlocks[mcid], blkCid)
		}
	}

	// Get messages that were processed in the parent tipset
	msgs, err := ar.ChainGetParentMessages(ctx, ts.Cids()[0])
	if err != nil {
		return nil, xerrors.Errorf("get parent messages: %w", err)
	}

	// Get receipts for parent messages
	rcpts, err := ar.ChainGetParentReceipts(ctx, ts.Cids()[0])
	if err != nil {
		return nil, xerrors.Errorf("get parent receipts: %w", err)
	}

	if len(rcpts) != len(msgs) {
		// logic error somewhere
		return nil, xerrors.Errorf("mismatching number of receipts: got %d wanted %d", len(rcpts), len(msgs))
	}

	// Start building a list of completed message with receipt
	emsgs := make([]*lens.ExecutedMessage, 0, len(msgs))

	// Create a skeleton vm just for calling ShouldBurn
	vmi, err := vm.NewVM(ctx, &vm.VMOpts{
		StateBase: pts.ParentState(),
		Epoch:     pts.Height(),
		Bstore:    &apiBlockstore{api: ar}, // sadly vm wraps this to turn it back into an adt.Store
	})
	if err != nil {
		return nil, xerrors.Errorf("creating temporary vm: %w", err)
	}

	for index, m := range msgs {

		em := &lens.ExecutedMessage{
			Cid:           m.Cid,
			Height:        pts.Height(),
			Message:       m.Message,
			Receipt:       rcpts[index],
			BlockHeader:   blockHeaders[messageBlocks[m.Cid][0]],
			Blocks:        messageBlocks[m.Cid],
			Index:         uint64(index),
			FromActorCode: getActorCode(m.Message.From),
			ToActorCode:   getActorCode(m.Message.To),
		}

		burn, err := vmi.ShouldBurn(parentStateTree, m.Message, rcpts[index].ExitCode)
		if err != nil {
			return nil, xerrors.Errorf("deciding whether should burn failed: %w", err)
		}

		em.GasOutputs = vm.ComputeGasOutputs(em.Receipt.GasUsed, em.Message.GasLimit, em.BlockHeader.ParentBaseFee, em.Message.GasFeeCap, em.Message.GasPremium, burn)
		emsgs = append(emsgs, em)
	}

	return emsgs, nil
}

type apiBlockstore struct {
	api interface {
		ChainReadObj(context.Context, cid.Cid) ([]byte, error)
		ChainHasObj(context.Context, cid.Cid) (bool, error)
	}
}

func (a *apiBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	data, err := a.api.ChainReadObj(context.Background(), c)
	if err != nil {
		return nil, err
	}

	return blocks.NewBlockWithCid(data, c)
}

func (a *apiBlockstore) Has(c cid.Cid) (bool, error) {
	return a.api.ChainHasObj(context.Background(), c)
}

func (a *apiBlockstore) DeleteBlock(c cid.Cid) error {
	return xerrors.Errorf("DeleteBlock not supported by apiBlockstore")
}

func (a *apiBlockstore) GetSize(c cid.Cid) (int, error) {
	data, err := a.api.ChainReadObj(context.Background(), c)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (a *apiBlockstore) Put(b blocks.Block) error {
	return xerrors.Errorf("Put not supported by apiBlockstore")
}

func (a *apiBlockstore) PutMany(bs []blocks.Block) error {
	return xerrors.Errorf("PutMany not supported by apiBlockstore")
}

func (a *apiBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, xerrors.Errorf("AllKeysChan not supported by apiBlockstore")
}

func (a *apiBlockstore) HashOnRead(enabled bool) {
}
